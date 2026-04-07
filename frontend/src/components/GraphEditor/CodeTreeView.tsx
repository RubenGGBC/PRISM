import React, { useState, useEffect, useMemo } from 'react';
import { TreeNode } from './TreeNode';
import { EditNodePanel } from './EditNodePanel';
import { GraphViz } from '../GraphViz';
import { Search, AlertCircle, FileText, Loader } from 'lucide-react';

interface TreeNodeData {
  id: string;
  name: string;
  type: string;
  file: string;
  line: number;
  signature?: string;
  comments?: string;
  tags?: string[];
  custom_metadata?: Record<string, string>;
  children?: TreeNodeData[];
}

interface CodeTreeViewProps {
  apiBaseUrl?: string;
}

export const CodeTreeView: React.FC<CodeTreeViewProps> = ({
  apiBaseUrl = 'http://localhost:8080',
}) => {
  const [treeData, setTreeData] = useState<TreeNodeData[]>([]);
  const [selectedNode, setSelectedNode] = useState<TreeNodeData | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());

  useEffect(() => {
    loadFiles();
  }, []);

  const loadFiles = async () => {
    try {
      setIsLoading(true);
      const response = await fetch(`${apiBaseUrl}/api/files`);
      if (!response.ok) throw new Error('Failed to load files');

      const data = await response.json();
      const files: TreeNodeData[] = (data.files || []).map((file: string) => ({
        id: file,
        name: file.split('\\').pop() || file,
        type: 'file',
        file,
        line: 0,
        children: [],
      }));

      setTreeData(files);
      setError(null);

      // Auto-load all file nodes so the graph shows connections from the start
      await Promise.all(
        (data.files || []).map((file: string) =>
          fetch(`${apiBaseUrl}/api/file/nodes?file=${encodeURIComponent(file)}`)
            .then((r) => r.json())
            .then((d) => {
              const nodes = d.nodes || [];
              setTreeData((prev) =>
                prev.map((f) =>
                  f.id === file
                    ? {
                        ...f,
                        children: nodes.map((node: any) => ({
                          id: node.id,
                          name: node.name,
                          type: node.type,
                          file: node.file,
                          line: node.line,
                          signature: node.signature,
                          comments: node.comments,
                          tags: node.tags,
                          custom_metadata: node.custom_metadata,
                          children: [],
                        })),
                      }
                    : f
                )
              );
            })
            .catch(() => {/* silently ignore per-file errors */})
        )
      );
    } catch (err) {
      setError(`Error loading files: ${err}`);
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  const loadFileNodes = async (filePath: string) => {
    try {
      const response = await fetch(
        `${apiBaseUrl}/api/file/nodes?file=${encodeURIComponent(filePath)}`
      );
      if (!response.ok) throw new Error('Failed to load file nodes');

      const data = await response.json();
      const nodes = data.nodes || [];

      // Update tree with loaded nodes
      setTreeData((prevData) =>
        prevData.map((file) =>
          file.id === filePath
            ? {
                ...file,
                children: nodes.map((node: any) => ({
                  id: node.id,
                  name: node.name,
                  type: node.type,
                  file: node.file,
                  line: node.line,
                  signature: node.signature,
                  comments: node.comments,
                  tags: node.tags,
                  custom_metadata: node.custom_metadata,
                  children: [],
                })),
              }
            : file
        )
      );
    } catch (err) {
      console.error('Error loading file nodes:', err);
    }
  };

  const handleSelectNode = async (node: TreeNodeData) => {
    try {
      const response = await fetch(
        `${apiBaseUrl}/api/node/full?id=${encodeURIComponent(node.id)}`
      );
      if (!response.ok) throw new Error('Failed to load node');

      const data = await response.json();
      setSelectedNode(data.node);
    } catch (err) {
      console.error('Error loading node:', err);
      setSelectedNode(node);
    }
  };

  const handleUpdateNode = async (nodeId: string, data: any) => {
    try {
      setIsSaving(true);
      const response = await fetch(`${apiBaseUrl}/api/node/update?id=${encodeURIComponent(nodeId)}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(data),
      });

      if (!response.ok) throw new Error('Failed to update node');

      if (selectedNode) {
        handleSelectNode(selectedNode);
      }

      setError(null);
    } catch (err) {
      setError(`Error updating node: ${err}`);
      console.error(err);
    } finally {
      setIsSaving(false);
    }
  };

  const filteredTree = treeData.filter((node) =>
    node.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    node.file.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const graphNodes = useMemo(() =>
    treeData.flatMap((file) => [
      { id: file.id, name: file.name, type: file.type },
      ...(file.children || []).map((child) => ({ id: child.id, name: child.name, type: child.type })),
    ]),
    [treeData]
  );

  const graphEdges = useMemo(() =>
    treeData.flatMap((file) =>
      (file.children || []).map((child) => ({ source: file.id, target: child.id }))
    ),
    [treeData]
  );

  return (
    <div className="flex h-full bg-slate-900">
      {/* Left Sidebar - Tree */}
      <div className="w-64 shrink-0 border-r border-slate-700/50 flex flex-col bg-slate-950 backdrop-blur-sm">
        {/* Search Bar */}
        <div className="p-4 border-b border-slate-700/50 bg-slate-950/50">
          <div className="relative group">
            <Search size={16} className="absolute left-3 top-3 text-slate-500 group-focus-within:text-emerald-400 transition" />
            <input
              type="text"
              placeholder="Search files..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-4 py-2.5 bg-slate-900 border border-slate-700 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 text-sm transition"
            />
          </div>
        </div>

        {/* Error Alert */}
        {error && (
          <div className="m-3 p-3 bg-red-950/40 border border-red-900/50 rounded-lg flex gap-2 items-start">
            <AlertCircle size={16} className="text-red-400 mt-0.5 flex-shrink-0" />
            <p className="text-red-300 text-xs">{error}</p>
          </div>
        )}

        {/* Tree Content */}
        <div className="flex-1 overflow-y-auto px-2 py-3">
          {isLoading ? (
            <div className="flex flex-col items-center justify-center h-full gap-3">
              <Loader size={20} className="text-emerald-400 animate-spin" />
              <p className="text-slate-400 text-sm">Loading files...</p>
            </div>
          ) : filteredTree.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full gap-3 text-center px-4">
              <FileText size={24} className="text-slate-600" />
              <p className="text-slate-400 text-sm">No files found</p>
              <p className="text-slate-500 text-xs">{searchQuery ? 'Try a different search term' : 'Your code graph is loading'}</p>
            </div>
          ) : (
            <div className="space-y-0.5">
              {filteredTree.map((node) => (
                <TreeNode
                  key={node.id}
                  node={node}
                  onSelectNode={handleSelectNode}
                  onUpdateNode={handleUpdateNode}
                  selectedNodeId={selectedNode?.id}
                  onExpandFile={node.type === 'file' ? () => loadFileNodes(node.id) : undefined}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer Stats */}
        <div className="border-t border-slate-700/50 px-4 py-3 text-xs text-slate-500 bg-slate-950/50">
          <p>{filteredTree.length} files • {treeData.length} total</p>
        </div>
      </div>

      {/* Center - Graph Visualization */}
      <div className="flex-1 bg-slate-900 overflow-hidden">
        <GraphViz
          nodes={graphNodes}
          edges={graphEdges}
          onNodeClick={(nodeId) => {
            const flat = treeData.flatMap((f) => [f, ...(f.children || [])]);
            const found = flat.find((n) => n.id === nodeId);
            if (found) handleSelectNode(found);
          }}
        />
      </div>

      {/* Right Panel - Edit Panel (only when node selected) */}
      {selectedNode && (
        <div className="w-80 shrink-0 border-l border-slate-700/50 bg-slate-900 flex flex-col">
          <EditNodePanel
            node={selectedNode}
            onUpdate={handleUpdateNode}
            onClose={() => setSelectedNode(null)}
            isSaving={isSaving}
          />
        </div>
      )}
    </div>
  );
};
