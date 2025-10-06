"use client";

import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { AlertTriangle } from "lucide-react";
import { Flow } from "./types";

interface DeleteConfirmModalProps {
  isOpen: boolean;
  onClose: () => void;
  flow: Flow | null;
  onConfirm: (flowId: string) => void;
}

export function DeleteConfirmModal({ isOpen, onClose, flow, onConfirm }: DeleteConfirmModalProps) {
  const handleConfirm = () => {
    if (flow) {
      onConfirm(flow.id);
      onClose();
    }
  };

  if (!flow) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[400px]">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-destructive/10">
              <AlertTriangle className="h-5 w-5 text-destructive" />
            </div>
            <div>
              <DialogTitle>Delete Protection Flow</DialogTitle>
              <DialogDescription>
                This action cannot be undone.
              </DialogDescription>
            </div>
          </div>
        </DialogHeader>

        <div className="py-4">
          <p className="text-sm text-muted-foreground">
            Are you sure you want to delete the protection flow <strong>"{flow.name}"</strong>?
            This will permanently remove the flow and all associated configuration.
          </p>
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleConfirm}
          >
            Delete Flow
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
