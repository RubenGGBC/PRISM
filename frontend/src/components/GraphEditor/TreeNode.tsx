import React, { useState } from 'react';
import { TagBadge } from './TagBadge';
import { ChevronRight, ChevronDown } from 'lucide-react';

interface TreeNodeData {
  id: string;
  name: string;
  type: string;
  file: string;
  line: number;
  comments?: string;
  tags?: string[];
  custom_metadata?: Record<string, string>;
  children?: TreeNodeData[];
}

interface TreeNodeProps {
  node: TreeNodeData;
  onSelectNode: (node: TreeNodeData) => void;
  onUpdateNode: (nodeId: string, data: any) => void;
  selectedNodeId?: string;
  isEditing?: boolean;
  editingNodeId?: string;
  onExpandFile?: () => void;
}

export const TreeNode: React.FC<TreeNodeProps> = ({
  node,
  onSelectNode,
  onUpdateNode,
  selectedNodeId,
  isEditing = false,
  editingNodeId,
  onExpandFile,
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const hasChildren = node.children && node.children.length > 0;
  const isSelected = selectedNodeId === node.id;

  const toggleExpand = async () => {
    const newExpanded = !isExpanded;
    setIsExpanded(newExpanded);
    // Load file nodes when expanding a file for the first time
    if (newExpanded && node.type === 'file' && hasChildren === false && onExpandFile) {
      await onExpandFile();
    }
  };

  const handleNodeClick = () => {
    onSelectNode(node);
  };

  return (
    <div className="select-none">
      <div
        className={`flex items-center gap-2 px-2 py-1.5 rounded-md cursor-pointer transition-all group ${
          isSelected
            ? 'bg-emerald-500/20 border border-emerald-500/30'
            : 'hover:bg-slate-800/50 border border-transparent hover:border-slate-700/30'
        }`}
        onClick={handleNodeClick}
      >
        {/* Expand/Collapse Button */}
        {hasChildren && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              toggleExpand();
            }}
            className="p-0 w-5 h-5 flex items-center justify-center text-slate-500 group-hover:text-emerald-400 transition"
          >
            {isExpanded ? (
              <ChevronDown size={16} />
            ) : (
              <ChevronRight size={16} />
            )}
          </button>
        )}
        {!hasChildren && <div className="w-5" />}

        {/* Node Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className={`font-medium text-sm ${
              isSelected ? 'text-emerald-300' : 'text-slate-200'
            }`}>
              {node.name}
            </span>
            <span className="text-xs bg-slate-800/70 text-slate-400 px-2 py-0.5 rounded border border-slate-700/50">
              {node.type}
            </span>
            {node.tags && node.tags.length > 0 && (
              <div className="flex gap-1">
                {node.tags.slice(0, 2).map((tag) => (
                  <TagBadge key={tag} tag={tag} />
                ))}
                {node.tags.length > 2 && (
                  <span className="text-xs text-slate-500">
                    +{node.tags.length - 2}
                  </span>
                )}
              </div>
            )}
          </div>
          <div className="text-xs text-slate-500 font-mono mt-0.5">
            {node.file}:{node.line}
          </div>
          {node.comments && (
            <div className="text-xs text-slate-400 mt-1 italic max-w-xs truncate">
              "{node.comments.substring(0, 50)}{node.comments.length > 50 ? '...' : ''}"
            </div>
          )}
        </div>
      </div>

      {/* Children */}
      {hasChildren && isExpanded && (
        <div className="ml-3 border-l border-slate-700/30 pl-1 py-0.5">
          {node.children!.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              onSelectNode={onSelectNode}
              onUpdateNode={onUpdateNode}
              selectedNodeId={selectedNodeId}
              editingNodeId={editingNodeId}
            />
          ))}
        </div>
      )}
    </div>
  );
};
