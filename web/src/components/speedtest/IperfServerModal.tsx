/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogClose,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/Button";

interface IperfServerModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (name: string) => void;
  title: string;
  message: string;
  confirmText: string;
  confirmStyle?: "danger" | "primary";
  serverDetails?: {
    host: string;
    port: string;
  };
}

export function IperfServerModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText,
  confirmStyle = "primary",
  serverDetails,
}: IperfServerModalProps) {
  const [serverName, setServerName] = useState("");

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          setServerName("");
          onClose();
        }
      }}
    >
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>

        <div className="grid gap-4">
          {serverDetails ? (
            <>
              <p className="text-muted-foreground text-sm">{message}</p>
              <div className="grid gap-3">
                <Label htmlFor="serverName">Server Name</Label>
                <Input
                  id="serverName"
                  value={serverName}
                  onChange={(e) => setServerName(e.target.value)}
                  placeholder="Enter a name for this server"
                  autoFocus
                />
              </div>
              <div className="grid gap-3">
                <Label>Server Host</Label>
                <Input
                  value={serverDetails.host}
                  placeholder="iperf.example.com"
                  readOnly
                />
              </div>
              <div className="grid gap-3">
                <Label>Server Port</Label>
                <Input value={serverDetails.port} readOnly />
              </div>
            </>
          ) : (
            <p className="text-muted-foreground text-sm">{message}</p>
          )}
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button
            disabled={serverDetails && !serverName.trim()}
            variant={confirmStyle === "danger" ? "destructive" : "default"}
            onClick={() => {
              onConfirm(serverName);
              setServerName("");
              onClose();
            }}
          >
            {confirmText}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
