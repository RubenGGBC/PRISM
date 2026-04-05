import React, { useEffect, useState } from 'react'
import { useWebSocket } from '../hooks/useWebSocket'

interface FileTreeProps {
  onFileSelect: (file: string) => void
}

export function FileTree({ onFileSelect }: FileTreeProps) {
  const [files, setFiles] = useState<string[]>([])
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  useEffect(() => {
    fetch('/api/files')
      .then(r => r.json())
      .then(d => setFiles(d.files))
  }, [])

  const toggleDir = (dir: string) => {
    const newExpanded = new Set(expanded)
    if (newExpanded.has(dir)) {
      newExpanded.delete(dir)
    } else {
      newExpanded.add(dir)
    }
    setExpanded(newExpanded)
  }

  return (
    <div className="bg-gray-900 text-white p-4 overflow-auto h-full">
      <h2 className="text-lg font-bold mb-4">Files</h2>
      {files.map(file => (
        <div key={file}>
          <button
            onClick={() => onFileSelect(file)}
            className="block w-full text-left px-2 py-1 hover:bg-gray-800 rounded"
          >
            📄 {file}
          </button>
        </div>
      ))}
    </div>
  )
}
