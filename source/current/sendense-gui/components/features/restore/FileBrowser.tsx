"use client";

import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Folder, File, Download, Archive, Search, Check } from "lucide-react";
import { useFiles } from "@/src/features/restore/hooks/useRestore";
import { BreadcrumbNav } from "./BreadcrumbNav";
import { FileRow } from "./FileRow";
import type { FileInfo } from "@/src/features/restore/types";

interface FileBrowserProps {
  mountId: string;
  currentPath: string;
  onNavigate: (path: string) => void;
}

export function FileBrowser({ mountId, currentPath, onNavigate }: FileBrowserProps) {
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState("");

  const { data: filesData, isLoading, error } = useFiles(mountId, currentPath);
  const files = filesData?.files || [];

  // Filter files based on search
  const filteredFiles = useMemo(() => {
    if (!searchQuery) return files;
    return files.filter(file =>
      file.name.toLowerCase().includes(searchQuery.toLowerCase())
    );
  }, [files, searchQuery]);

  const handleSelectFile = (fileName: string, checked: boolean) => {
    const newSelected = new Set(selectedFiles);
    if (checked) {
      newSelected.add(fileName);
    } else {
      newSelected.delete(fileName);
    }
    setSelectedFiles(newSelected);
  };

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedFiles(new Set(filteredFiles.map(f => f.name)));
    } else {
      setSelectedFiles(new Set());
    }
  };

  const handleDownloadFile = (filePath: string) => {
    const url = `/api/v1/restore/${mountId}/download?path=${encodeURIComponent(filePath)}`;
    window.open(url, '_blank');
  };

  const handleDownloadSelected = () => {
    // For multiple files, we'd need to create a ZIP on the server
    // For now, download each file individually
    selectedFiles.forEach(fileName => {
      const filePath = currentPath === '/' ? `/${fileName}` : `${currentPath}/${fileName}`;
      handleDownloadFile(filePath);
    });
    setSelectedFiles(new Set());
  };

  const handleDownloadFolder = (folderPath: string) => {
    const url = `/api/v1/restore/${mountId}/download-directory?path=${encodeURIComponent(folderPath)}&format=zip`;
    window.open(url, '_blank');
  };

  const handleDoubleClick = (file: FileInfo) => {
    if (file.type === 'directory') {
      const newPath = currentPath === '/' ? `/${file.name}` : `${currentPath}/${file.name}`;
      onNavigate(newPath);
    }
  };

  if (error) {
    return (
      <Card>
        <CardContent className="p-6 text-center">
          <div className="text-destructive">Failed to load files: {error.message}</div>
          <Button
            variant="outline"
            onClick={() => window.location.reload()}
            className="mt-4"
          >
            Retry
          </Button>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Folder className="h-5 w-5" />
          Browse Files
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Breadcrumb Navigation */}
        <BreadcrumbNav path={currentPath} onNavigate={onNavigate} />

        {/* Search and Controls */}
        <div className="flex items-center gap-4">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search files..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>

          {selectedFiles.size > 0 && (
            <Button onClick={handleDownloadSelected} className="gap-2">
              <Download className="h-4 w-4" />
              Download Selected ({selectedFiles.size})
            </Button>
          )}
        </div>

        {/* File List */}
        <div className="border rounded-lg">
          {/* Header */}
          <div className="flex items-center px-4 py-3 border-b bg-muted/50">
            <Checkbox
              checked={filteredFiles.length > 0 && selectedFiles.size === filteredFiles.length}
              indeterminate={selectedFiles.size > 0 && selectedFiles.size < filteredFiles.length}
              onCheckedChange={handleSelectAll}
              className="mr-3"
            />
            <div className="grid grid-cols-12 gap-4 flex-1 text-sm font-medium text-muted-foreground">
              <div className="col-span-6">Name</div>
              <div className="col-span-2 text-right">Size</div>
              <div className="col-span-3">Modified</div>
              <div className="col-span-1">Actions</div>
            </div>
          </div>

          {/* Files */}
          <div className="max-h-96 overflow-auto">
            {isLoading ? (
              <div className="flex items-center justify-center py-8">
                <div className="text-muted-foreground">Loading files...</div>
              </div>
            ) : filteredFiles.length === 0 ? (
              <div className="flex items-center justify-center py-8">
                <div className="text-muted-foreground">
                  {searchQuery ? 'No files match your search' : 'No files in this directory'}
                </div>
              </div>
            ) : (
              filteredFiles.map((file) => (
                <FileRow
                  key={file.name}
                  file={file}
                  isSelected={selectedFiles.has(file.name)}
                  onSelect={handleSelectFile}
                  onDownload={handleDownloadFile}
                  onDownloadFolder={handleDownloadFolder}
                  onDoubleClick={handleDoubleClick}
                />
              ))
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
