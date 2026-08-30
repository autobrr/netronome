# ADR-0001: Notification message templates on the rule

Date: 2026-08-30
Status: accepted
Issue: #157

## Context

Every notification message body is built in Go code in `Notifier`, with
`fmt.Sprintf` and `strings.Builder`. There are 15 event types in three categories.
A user cannot change any message.

This causes two problems. A user cannot add the information that they want or
remove the information that they do not want. And the messages contain Discord
Markdown (`**text**`) and arrow characters, which Discord shows correctly but
ntfy and other plain-text targets do not.

The alternative to templates is a formatter that knows each target. That needs one
branch for each of the 25 or more shoutrrr services, it is never complete, and the
project maintains it forever.

## Decision

A user can give a template for the message body.

**The template belongs to the rule.** A rule joins one channel to one event, so a
Discord rule and an ntfy rule for the same event can have different messages. This
is the only place that solves the format problem. The rule already holds the
threshold value and the threshold operator, so it is the established place for a
decision that a user makes for one destination and one event.

**A rule with no template uses the Go formatter.** The column is nullable and NULL
is the default. The migration writes nothing to the rules that exist. A user who
does not want this feature sees no change.

**The engine is `text/template` with sprig.** This is the syntax that autobrr uses
for its filters (`internal/domain/macros.go`), so the users of both programs write
one syntax. Sprig gives the number formatting and the conditions without a
function library in this repository.

**The template data is one flat structure for all events.** This follows the
`Macro` type in autobrr. The fields that do not apply to the event of the moment
stay at their zero value. Each value that has a unit appears twice: the raw number
for a comparison, and a formatted string with the unit for the common case.

**Three sprig functions are removed:** `env`, `expandenv`, and `getHostByName`.
They read the environment of the server, and the result of a template goes to an
external service. `sprig.HermeticTxtFuncMap()` removes these three, but it also
removes `now`, `date`, and `dateInZone`, which a message needs. Therefore the code
deletes three keys from `sprig.TxtFuncMap()`.

**A template that does not parse is refused when the user saves it.** The error
goes to the editor, where the person who wrote the template can see it. An error
during the render of a template that did parse does not stop the notification: the
notifier sends the message of the Go formatter and writes the error in the log. An
alert with a bad message is better than no alert.

## Consequences

The Go formatters stay. They are the default and the fallback, and they are the
code to change when a default message is bad.

The editor has a button that fills the text field with a template that gives the
same result as the Go formatter. That template string is a copy of the logic of
the formatter, and the two can become different without a signal. This is accepted.
The alternative, to make the templates the only source and to write them to the
rules of the current users in the migration, puts the notifications of people who
did not ask for this feature at risk of a fault in a migration.

The title of a notification is not a template. `notificationTitle` keeps its
behavior. If a user asks for a title template later, it is a second field with a
second validation, and this decision does not block it.
