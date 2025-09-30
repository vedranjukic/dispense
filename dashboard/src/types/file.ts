export interface FileItem {
  path: string;
  status: 'added' | 'modified' | 'deleted';
  size?: number;
  lastModified?: string;
}

export interface ModifiedFilesListProps {
  sandboxId: string;
  onFileSelect?: (filePath: string) => void;
}

export interface FileSystemWatcher {
  sandboxId: string;
  files: FileItem[];
  lastUpdate: number;
}