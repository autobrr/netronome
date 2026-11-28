import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

// Mark a code span that starts a list item ("- `key`: text") as a term, so CSS
// can draw it without the inline code box.
function remarkListTerms() {
  const walk = (node) => {
    if (node.type === "listItem") {
      const first = node.children[0]?.children?.[0];
      if (node.children[0]?.type === "paragraph" && first?.type === "inlineCode") {
        first.data = { hProperties: { className: ["term"] } };
      }
    }
    node.children?.forEach(walk);
  };
  return walk;
}

export default defineConfig({
  site: "https://netrono.me",
  markdown: { remarkPlugins: [remarkListTerms] },
  integrations: [
    starlight({
      title: "Netronome",
      description: "Network performance testing and monitoring.",
      logo: { src: "./src/assets/logo.png" },
      favicon: "/favicon.ico",
      customCss: ["./src/styles/netronome.css"],
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/autobrr/netronome" },
        { icon: "discord", label: "Discord", href: "https://discord.gg/WehFCZxq5B" },
      ],
      editLink: { baseUrl: "https://github.com/autobrr/netronome/edit/develop/documentation/" },
      sidebar: [
        { label: "Getting started", items: [{ autogenerate: { directory: "getting-started" } }] },
        { label: "Configuration", items: [{ autogenerate: { directory: "configuration" } }] },
        { label: "Monitoring", items: [{ autogenerate: { directory: "monitoring" } }] },
        { label: "Reference", items: [{ autogenerate: { directory: "reference" } }] },
        { label: "Help", items: [{ autogenerate: { directory: "help" } }] },
      ],
    }),
  ],
});
