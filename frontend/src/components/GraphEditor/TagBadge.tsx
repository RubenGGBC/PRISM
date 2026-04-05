import React from 'react';

interface TagBadgeProps {
  tag: string;
  onRemove?: (tag: string) => void;
  editable?: boolean;
}

export const TagBadge: React.FC<TagBadgeProps> = ({ tag, onRemove, editable = false }) => {
  return (
    <div className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-emerald-500/15 text-emerald-300 border border-emerald-500/30 rounded-full text-xs font-medium hover:bg-emerald-500/20 transition group">
      <span>{tag}</span>
      {editable && onRemove && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onRemove(tag);
          }}
          className="ml-0.5 text-emerald-400 hover:text-emerald-200 font-bold opacity-0 group-hover:opacity-100 transition"
          title="Remove tag"
        >
          ×
        </button>
      )}
    </div>
  );
};
