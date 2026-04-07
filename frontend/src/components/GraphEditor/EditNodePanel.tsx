import React, { useState, useEffect } from 'react';
import { TagBadge } from './TagBadge';
import { X, FileCode, MessageSquare, Tag, Settings, CheckCircle, Loader } from 'lucide-react';

interface NodeData {
  id: string;
  name: string;
  type: string;
  file: string;
  line: number;
  signature?: string;
  comments?: string;
  tags?: string[];
  custom_metadata?: Record<string, string>;
  why?: string;
  status?: string;
  entry_point?: boolean;
  known_bug?: string;
}

interface EditNodePanelProps {
  node: NodeData | null;
  onUpdate: (nodeId: string, data: any) => void;
  onClose: () => void;
  isSaving?: boolean;
}

export const EditNodePanel: React.FC<EditNodePanelProps> = ({
  node,
  onUpdate,
  onClose,
  isSaving = false,
}) => {
  const [comments, setComments] = useState('');
  const [tags, setTags] = useState<string[]>([]);
  const [newTag, setNewTag] = useState('');
  const [metadata, setMetadata] = useState<Record<string, string>>({});
  const [metaKey, setMetaKey] = useState('');
  const [metaValue, setMetaValue] = useState('');
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [why, setWhy] = useState('');
  const [status, setStatus] = useState('stable');
  const [entryPoint, setEntryPoint] = useState(false);
  const [knownBug, setKnownBug] = useState('');

  useEffect(() => {
    if (node) {
      setComments(node.comments || '');
      setTags(node.tags || []);
      setMetadata(node.custom_metadata || {});
      setNewTag('');
      setMetaKey('');
      setMetaValue('');
      setSaveSuccess(false);
      setWhy(node.why || '');
      setStatus(node.status || 'stable');
      setEntryPoint(node.entry_point || false);
      setKnownBug(node.known_bug || '');
    }
  }, [node]);

  const handleSave = async () => {
    if (!node) return;

    onUpdate(node.id, {
      comments,
      tags,
      custom_metadata: metadata,
      why,
      status,
      entry_point: entryPoint,
      known_bug: knownBug,
    });

    setSaveSuccess(true);
    setTimeout(() => setSaveSuccess(false), 2000);
  };

  const handleAddTag = () => {
    if (newTag.trim() && !tags.includes(newTag.trim())) {
      setTags([...tags, newTag.trim()]);
      setNewTag('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setTags(tags.filter((t) => t !== tagToRemove));
  };

  const handleAddMetadata = () => {
    if (metaKey.trim() && metaValue.trim()) {
      setMetadata({
        ...metadata,
        [metaKey.trim()]: metaValue.trim(),
      });
      setMetaKey('');
      setMetaValue('');
    }
  };

  const handleRemoveMetadata = (key: string) => {
    const newMeta = { ...metadata };
    delete newMeta[key];
    setMetadata(newMeta);
  };

  if (!node) {
    return null;
  }

  return (
    <div className="w-full h-full bg-slate-900 border-l border-slate-700/50 overflow-y-auto flex flex-col">
      {/* Sticky Header */}
      <div className="sticky top-0 z-10 bg-slate-900/95 backdrop-blur border-b border-slate-700/50 px-6 py-5 flex justify-between items-start gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <div className="p-1.5 rounded bg-emerald-500/10">
              <FileCode size={14} className="text-emerald-400" />
            </div>
            <h2 className="text-lg font-semibold text-slate-100 truncate">{node.name}</h2>
            <span className="px-2 py-0.5 text-xs bg-slate-800 text-slate-300 rounded">{node.type}</span>
          </div>
          <p className="text-xs text-slate-500 font-mono">{node.file}:{node.line}</p>
        </div>
        <button
          onClick={onClose}
          className="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        >
          <X size={18} />
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 px-6 py-6 space-y-6 overflow-y-auto">
        {/* Signature */}
        {node.signature && (
          <div className="space-y-2">
            <label className="text-xs font-semibold text-slate-300 uppercase tracking-wide">Signature</label>
            <pre className="bg-slate-800/50 border border-slate-700/50 p-3 rounded-lg text-xs text-slate-200 overflow-x-auto font-mono">
              {node.signature}
            </pre>
          </div>
        )}

        {/* Comments */}
        <div className="space-y-2">
          <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 uppercase tracking-wide">
            <MessageSquare size={14} className="text-emerald-400" />
            Comments
          </label>
          <textarea
            value={comments}
            onChange={(e) => setComments(e.target.value)}
            placeholder="Add your notes about this function..."
            className="w-full h-24 px-4 py-3 bg-slate-800/50 border border-slate-700 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 text-sm transition resize-none"
          />
        </div>

        {/* Why does this exist */}
        <div className="space-y-2">
          <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 uppercase tracking-wide">
            <span className="text-emerald-400">?</span>
            Por qué existe
          </label>
          <textarea
            value={why}
            onChange={(e) => setWhy(e.target.value)}
            placeholder="Contexto de negocio: por qué existe esta función..."
            className="w-full h-20 px-4 py-3 bg-slate-800/50 border border-slate-700 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 text-sm transition resize-none"
          />
        </div>

        {/* Status */}
        <div className="space-y-2">
          <label className="text-xs font-semibold text-slate-300 uppercase tracking-wide">Estado</label>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="w-full px-3 py-2 bg-slate-800/50 border border-slate-700 text-slate-100 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 text-sm"
          >
            <option value="stable">Stable</option>
            <option value="legacy">Legacy</option>
            <option value="deprecated">Deprecated</option>
            <option value="critical">Critical</option>
          </select>
        </div>

        {/* Entry point toggle */}
        <div className="flex items-center justify-between py-2">
          <label className="text-xs font-semibold text-slate-300 uppercase tracking-wide">Entry Point</label>
          <button
            onClick={() => setEntryPoint(!entryPoint)}
            className={`relative w-10 h-5 rounded-full transition ${entryPoint ? 'bg-emerald-500' : 'bg-slate-600'}`}
          >
            <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${entryPoint ? 'translate-x-5' : ''}`} />
          </button>
        </div>

        {/* Known bug */}
        <div className="space-y-2">
          <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 uppercase tracking-wide">
            <span className="text-red-400">⚠</span>
            Bug conocido
          </label>
          <textarea
            value={knownBug}
            onChange={(e) => setKnownBug(e.target.value)}
            placeholder="Describe bugs conocidos que Claude debe tener en cuenta..."
            className="w-full h-20 px-4 py-3 bg-slate-800/50 border border-slate-700/50 border-red-900/30 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-red-500 text-sm transition resize-none"
          />
        </div>

        {/* Tags */}
        <div className="space-y-3">
          <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 uppercase tracking-wide">
            <Tag size={14} className="text-emerald-400" />
            Tags
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={newTag}
              onChange={(e) => setNewTag(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter') {
                  handleAddTag();
                  e.preventDefault();
                }
              }}
              placeholder="Type a tag..."
              className="flex-1 px-3 py-2 bg-slate-800/50 border border-slate-700 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 text-sm transition"
            />
            <button
              onClick={handleAddTag}
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-sm font-medium transition"
            >
              Add
            </button>
          </div>
          {tags.length > 0 && (
            <div className="flex flex-wrap gap-2 pt-2">
              {tags.map((tag) => (
                <TagBadge
                  key={tag}
                  tag={tag}
                  onRemove={handleRemoveTag}
                  editable={true}
                />
              ))}
            </div>
          )}
        </div>

        {/* Custom Metadata */}
        <div className="space-y-3">
          <label className="flex items-center gap-2 text-xs font-semibold text-slate-300 uppercase tracking-wide">
            <Settings size={14} className="text-emerald-400" />
            Metadata
          </label>
          <div className="flex gap-2">
            <input
              type="text"
              value={metaKey}
              onChange={(e) => setMetaKey(e.target.value)}
              placeholder="Key..."
              className="flex-1 px-3 py-2 bg-slate-800/50 border border-slate-700 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 text-sm transition"
            />
            <input
              type="text"
              value={metaValue}
              onChange={(e) => setMetaValue(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter') {
                  handleAddMetadata();
                  e.preventDefault();
                }
              }}
              placeholder="Value..."
              className="flex-1 px-3 py-2 bg-slate-800/50 border border-slate-700 text-slate-100 placeholder-slate-500 rounded-lg focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 text-sm transition"
            />
            <button
              onClick={handleAddMetadata}
              className="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-sm font-medium transition"
            >
              Add
            </button>
          </div>
          {Object.entries(metadata).length > 0 && (
            <div className="space-y-2 pt-2">
              {Object.entries(metadata).map(([key, value]) => (
                <div
                  key={key}
                  className="flex justify-between items-center gap-2 bg-slate-800/30 border border-slate-700/50 px-3 py-2 rounded-lg group hover:bg-slate-800/50 transition"
                >
                  <div className="min-w-0 flex-1">
                    <span className="text-xs font-mono text-emerald-300">{key}</span>
                    <span className="text-slate-400 mx-2">=</span>
                    <span className="text-xs text-slate-300">{value}</span>
                  </div>
                  <button
                    onClick={() => handleRemoveMetadata(key)}
                    className="p-1 rounded text-slate-500 hover:text-red-400 hover:bg-red-950/20 opacity-0 group-hover:opacity-100 transition"
                  >
                    <X size={14} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Footer - Save Button */}
      <div className="sticky bottom-0 px-6 py-4 border-t border-slate-700/50 bg-slate-900/95 backdrop-blur">
        <button
          onClick={handleSave}
          disabled={isSaving}
          className={`w-full px-4 py-3 rounded-lg font-medium text-sm transition flex items-center justify-center gap-2 ${
            saveSuccess
              ? 'bg-emerald-600/20 text-emerald-300 border border-emerald-600/30'
              : isSaving
              ? 'bg-slate-700 text-slate-300 cursor-not-allowed'
              : 'bg-emerald-600 hover:bg-emerald-700 text-white'
          }`}
        >
          {isSaving ? (
            <>
              <Loader size={16} className="animate-spin" />
              Saving...
            </>
          ) : saveSuccess ? (
            <>
              <CheckCircle size={16} />
              Saved Successfully
            </>
          ) : (
            'Save Changes'
          )}
        </button>
      </div>
    </div>
  );
};
