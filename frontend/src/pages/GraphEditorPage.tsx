import React from 'react';
import { CodeTreeView } from '../components/GraphEditor/CodeTreeView';
import { Database, Sparkles } from 'lucide-react';

export const GraphEditorPage: React.FC = () => {
  return (
    <div className="h-screen flex flex-col bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950">
      {/* Background Grid Effect */}
      <div className="fixed inset-0 pointer-events-none opacity-5">
        <svg className="w-full h-full" xmlns="http://www.w3.org/2000/svg">
          <defs>
            <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
              <path d="M 40 0 L 0 0 0 40" fill="none" stroke="white" strokeWidth="0.5"/>
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill="url(#grid)" />
        </svg>
      </div>

      {/* Header */}
      <div className="relative border-b border-emerald-900/30 backdrop-blur-sm bg-slate-950/40">
        <div className="px-8 py-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-emerald-500/10 border border-emerald-500/30">
              <Database className="w-6 h-6 text-emerald-400" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-3xl font-bold bg-gradient-to-r from-emerald-200 to-emerald-400 bg-clip-text text-transparent">
                  Graph Editor
                </h1>
                <Sparkles className="w-5 h-5 text-emerald-400 opacity-60" />
              </div>
              <p className="text-sm text-slate-400 mt-1">Semantic code annotation & analysis</p>
            </div>
          </div>
          <div className="text-right text-xs text-slate-500">
            Connected to <span className="text-emerald-400 font-mono">localhost:8080</span>
          </div>
        </div>
      </div>

      {/* Editor */}
      <div className="flex-1 relative overflow-hidden">
        <CodeTreeView apiBaseUrl="http://localhost:8080" />
      </div>

      {/* Footer Accent */}
      <div className="h-1 bg-gradient-to-r from-emerald-600/0 via-emerald-500/50 to-emerald-600/0"></div>
    </div>
  );
};
