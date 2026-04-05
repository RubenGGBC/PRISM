import React, { useEffect, useRef } from 'react'
import * as monaco from 'monaco-editor'

interface CodeEditorProps {
  filename: string
  code: string
  language?: string
}

export function CodeEditor({ filename, code, language = 'typescript' }: CodeEditorProps) {
  const editorRef = useRef<HTMLDivElement>(null)
  const editorInstance = useRef<monaco.editor.IStandaloneCodeEditor | null>(null)

  useEffect(() => {
    if (!editorRef.current) return

    if (!editorInstance.current) {
      editorInstance.current = monaco.editor.create(editorRef.current, {
        value: code,
        language: language,
        theme: 'vs-dark',
        readOnly: true,
        minimap: { enabled: true },
      })
    } else {
      editorInstance.current.setValue(code)
      monaco.editor.setModelLanguage(editorInstance.current.getModel()!, language)
    }

    return () => {
      // Don't dispose on unmount to avoid memory leaks
    }
  }, [code, language])

  return (
    <div className="flex flex-col h-full bg-gray-900">
      <div className="px-4 py-2 bg-gray-800 text-white font-mono">
        {filename}
      </div>
      <div ref={editorRef} className="flex-1" />
    </div>
  )
}
