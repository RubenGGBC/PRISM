import React, { useState, useEffect } from 'react';
import { Plus, Trash2, Tag } from 'lucide-react';

interface Profile {
  id: string;
  name: string;
  description: string;
}

interface ProfilesProps {
  apiBaseUrl: string;
  selectedNodeId?: string;
}

export const Profiles: React.FC<ProfilesProps> = ({ apiBaseUrl, selectedNodeId }) => {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    loadProfiles();
  }, []);

  const loadProfiles = async () => {
    try {
      const res = await fetch(`${apiBaseUrl}/api/profiles`);
      const data = await res.json();
      setProfiles(data.profiles || []);
    } catch {}
  };

  const createProfile = async () => {
    if (!newName.trim()) return;
    await fetch(`${apiBaseUrl}/api/profiles`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: newName.trim(), description: newDesc.trim() }),
    });
    setNewName('');
    setNewDesc('');
    setCreating(false);
    loadProfiles();
  };

  const deleteProfile = async (id: string) => {
    await fetch(`${apiBaseUrl}/api/profiles?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
    loadProfiles();
  };

  const addNodeToProfile = async (profileId: string) => {
    if (!selectedNodeId) return;
    await fetch(`${apiBaseUrl}/api/profile/nodes?profile_id=${encodeURIComponent(profileId)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ node_id: selectedNodeId }),
    });
  };

  return (
    <div className="px-3 py-3 border-t border-slate-700/50">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-semibold text-slate-400 uppercase tracking-wide">Perfiles</span>
        <button
          onClick={() => setCreating(!creating)}
          className="p-1 rounded hover:bg-slate-800 text-slate-500 hover:text-emerald-400 transition"
        >
          <Plus size={14} />
        </button>
      </div>

      {creating && (
        <div className="space-y-1.5 mb-2">
          <input
            type="text"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Nombre (ej. auth)"
            className="w-full px-2 py-1.5 bg-slate-900 border border-slate-700 text-slate-100 placeholder-slate-500 rounded text-xs focus:outline-none focus:ring-1 focus:ring-emerald-500"
          />
          <input
            type="text"
            value={newDesc}
            onChange={(e) => setNewDesc(e.target.value)}
            placeholder="Descripción (opcional)"
            className="w-full px-2 py-1.5 bg-slate-900 border border-slate-700 text-slate-100 placeholder-slate-500 rounded text-xs focus:outline-none focus:ring-1 focus:ring-emerald-500"
          />
          <button
            onClick={createProfile}
            className="w-full py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded text-xs font-medium transition"
          >
            Crear perfil
          </button>
        </div>
      )}

      <div className="space-y-1">
        {profiles.map((p) => (
          <div key={p.id} className="flex items-center gap-1.5 group">
            <span className="flex-1 text-xs text-slate-300 truncate">{p.name}</span>
            {selectedNodeId && (
              <button
                onClick={() => addNodeToProfile(p.id)}
                title="Añadir nodo seleccionado"
                className="p-0.5 rounded opacity-0 group-hover:opacity-100 hover:bg-slate-700 text-slate-500 hover:text-emerald-400 transition"
              >
                <Tag size={12} />
              </button>
            )}
            <button
              onClick={() => deleteProfile(p.id)}
              className="p-0.5 rounded opacity-0 group-hover:opacity-100 hover:bg-slate-700 text-slate-500 hover:text-red-400 transition"
            >
              <Trash2 size={12} />
            </button>
          </div>
        ))}
        {profiles.length === 0 && (
          <p className="text-xs text-slate-600">Sin perfiles aún</p>
        )}
      </div>
    </div>
  );
};
