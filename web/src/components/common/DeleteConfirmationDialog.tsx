/*
 * Copyright (c) 2024-2025, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { motion } from "motion/react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { ExclamationTriangleIcon } from "@heroicons/react/24/outline";

interface DeleteConfirmationDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title?: string;
  itemName: string;
  description?: string;
  isDeleting?: boolean;
}

export const DeleteConfirmationDialog: React.FC<DeleteConfirmationDialogProps> = ({
  isOpen,
  onClose,
  onConfirm,
  title = "Delete Agent",
  itemName,
  description,
  isDeleting = false,
}) => {
  return (
    <Dialog open={isOpen} onOpenChange={(open) => {
      // Only allow closing if not currently deleting
      if (!open && !isDeleting) {
        onClose();
      }
    }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader className="space-y-3">
          <div className="flex items-center gap-3">
            <motion.div
              initial={{ scale: 0.8, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              transition={{ duration: 0.2 }}
              className="h-10 w-10 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center flex-shrink-0"
            >
              <ExclamationTriangleIcon className="h-6 w-6 text-red-600 dark:text-red-400" />
            </motion.div>
            <DialogTitle className="text-lg font-semibold text-gray-900 dark:text-white">
              {title}
            </DialogTitle>
          </div>
          <DialogDescription className="text-gray-600 dark:text-gray-400">
            {description || (
              <>
                Are you sure you want to delete <span className="font-medium text-gray-900 dark:text-white">{itemName}</span>? 
                This action cannot be undone.
              </>
            )}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="mt-6 flex flex-col-reverse sm:flex-row sm:justify-end gap-2">
          <Button
            variant="secondary"
            onClick={onClose}
            disabled={isDeleting}
            className="w-full sm:w-auto"
          >
            Cancel
          </Button>
          <Button
            variant="default"
            onClick={onConfirm}
            disabled={isDeleting}
            className="w-full sm:w-auto bg-red-600 hover:bg-red-700 text-white border-red-700 hover:border-red-800"
          >
            {isDeleting ? (
              <>
                <motion.div
                  animate={{ rotate: 360 }}
                  transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                  className="w-4 h-4 border-2 border-white border-t-transparent rounded-full mr-2"
                />
                Deleting...
              </>
            ) : (
              "Delete"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};