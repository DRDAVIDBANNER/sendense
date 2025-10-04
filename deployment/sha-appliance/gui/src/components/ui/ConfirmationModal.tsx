'use client';

import React from 'react';
import { Modal, Button } from 'flowbite-react';

export interface ConfirmationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  confirmColor?: 'success' | 'failure' | 'warning' | 'purple';
  dangerous?: boolean;
}

export const ConfirmationModal = React.memo(({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  confirmColor = 'failure',
  dangerous = false
}: ConfirmationModalProps) => {
  const handleConfirm = React.useCallback(() => {
    onConfirm();
    onClose();
  }, [onConfirm, onClose]);

  return (
    <Modal show={isOpen} onClose={onClose} size="md">
      <Modal.Header>
        <div className="flex items-center space-x-2">
          {dangerous && (
            <span className="text-red-500 text-xl">⚠️</span>
          )}
          <span>{title}</span>
        </div>
      </Modal.Header>
      <Modal.Body>
        <div className="space-y-4">
          <p className="text-gray-700 dark:text-gray-300">
            {message}
          </p>
          {dangerous && (
            <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
              <p className="text-sm text-red-700 dark:text-red-300">
                <strong>⚠️ Warning:</strong> This action cannot be undone. Please proceed with caution.
              </p>
            </div>
          )}
        </div>
      </Modal.Body>
      <Modal.Footer>
        <div className="flex space-x-2 ml-auto">
          <Button color="gray" onClick={onClose}>
            {cancelText}
          </Button>
          <Button color={confirmColor} onClick={handleConfirm}>
            {confirmText}
          </Button>
        </div>
      </Modal.Footer>
    </Modal>
  );
});

ConfirmationModal.displayName = 'ConfirmationModal';
