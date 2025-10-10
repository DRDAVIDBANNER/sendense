"use client";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Download, Archive, Folder, File } from "lucide-react";
import type { FileInfo } from "@/src/features/restore/types";

interface FileRowProps {
  file: FileInfo;
  isSelected: boolean;
  onSelect: (fileName: string, checked: boolean) => void;
  onDownload: (filePath: string) => void;
  onDownloadFolder: (folderPath: string) => void;
  onDoubleClick: (file: FileInfo) => void;
}

export function FileRow({
  file,
  isSelected,
  onSelect,
  onDownload,
  onDownloadFolder,
  onDoubleClick
}: FileRowProps) {
  const formatSize = (bytes: number) => {
    if (bytes === 0) return '-';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
  };

  const formatDate = (dateString: string) => {
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return dateString;
    }
  };

  const handleDownload = () => {
    if (file.type === 'directory') {
      onDownloadFolder(file.path);
    } else {
      onDownload(file.path);
    }
  };

  return (
    <div
      className="flex items-center px-4 py-3 border-b border-border hover:bg-muted/50 cursor-pointer"
      onDoubleClick={() => onDoubleClick(file)}
    >
      <Checkbox
        checked={isSelected}
        onCheckedChange={(checked) => onSelect(file.name, checked as boolean)}
        className="mr-3"
      />

      <div className="grid grid-cols-12 gap-4 flex-1 items-center">
        {/* Name */}
        <div className="col-span-6 flex items-center gap-3">
          {file.type === 'directory' ? (
            <Folder className="h-4 w-4 text-blue-500" />
          ) : (
            <File className="h-4 w-4 text-muted-foreground" />
          )}
          <span className="truncate font-medium">{file.name}</span>
        </div>

        {/* Size */}
        <div className="col-span-2 text-right text-sm text-muted-foreground">
          {file.type === 'directory' ? '-' : formatSize(file.size)}
        </div>

        {/* Modified */}
        <div className="col-span-3 text-sm text-muted-foreground">
          {formatDate(file.modified_time)}
        </div>

        {/* Actions */}
        <div className="col-span-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => {
              e.stopPropagation();
              handleDownload();
            }}
            className="h-8 w-8 p-0"
          >
            {file.type === 'directory' ? (
              <Archive className="h-4 w-4" />
            ) : (
              <Download className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}

