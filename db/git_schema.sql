-- ============================================
-- Git History Tables
-- ============================================

-- Historia de git para cada función/archivo
CREATE TABLE IF NOT EXISTS git_function_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Identificación
    file_path TEXT NOT NULL,
    function_name TEXT NOT NULL,

    -- Fechas clave
    created_date TIMESTAMP,
    created_by TEXT,
    last_modified TIMESTAMP,

    -- Métricas
    commits_count INTEGER DEFAULT 0,
    bug_fixes INTEGER DEFAULT 0,
    breaking_changes INTEGER DEFAULT 0,
    refactors_count INTEGER DEFAULT 0,
    features_count INTEGER DEFAULT 0,

    -- Contexto
    is_active BOOLEAN DEFAULT 1,
    stability_score INTEGER DEFAULT 50,
    risk_assessment TEXT DEFAULT 'UNKNOWN',

    -- Datos JSON
    evolution_json TEXT,       -- JSON array de EvolutionEvent
    team_owners_json TEXT,     -- JSON object {author: percent}

    -- Metadata
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    UNIQUE(file_path, function_name)
);

-- Team context: quién sabe qué
CREATE TABLE IF NOT EXISTS git_team_context (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    function_id INTEGER NOT NULL,
    author TEXT NOT NULL,

    -- Estadísticas
    commits INTEGER DEFAULT 0,
    contribution_percent REAL DEFAULT 0.0,
    expertise_level TEXT DEFAULT 'minimal', -- expert, familiar, minimal

    -- Fechas
    first_seen TIMESTAMP,
    last_seen TIMESTAMP,
    days_since_active INTEGER DEFAULT 0,

    -- Metadata
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (function_id) REFERENCES git_function_history(id) ON DELETE CASCADE,
    UNIQUE(function_id, author)
);

-- ============================================
-- Índices para performance
-- ============================================

CREATE INDEX IF NOT EXISTS idx_git_file_function
    ON git_function_history(file_path, function_name);

CREATE INDEX IF NOT EXISTS idx_git_file
    ON git_function_history(file_path);

CREATE INDEX IF NOT EXISTS idx_git_risk
    ON git_function_history(risk_assessment);

CREATE INDEX IF NOT EXISTS idx_git_active
    ON git_function_history(is_active);

CREATE INDEX IF NOT EXISTS idx_team_function
    ON git_team_context(function_id);

CREATE INDEX IF NOT EXISTS idx_team_author
    ON git_team_context(author);

CREATE INDEX IF NOT EXISTS idx_team_expertise
    ON git_team_context(expertise_level);

-- ============================================
-- Vistas útiles
-- ============================================

-- Funciones más riesgosas
CREATE VIEW IF NOT EXISTS high_risk_functions AS
SELECT file_path, function_name, risk_assessment, commits_count, bug_fixes
FROM git_function_history
WHERE risk_assessment = 'HIGH'
ORDER BY bug_fixes DESC;

-- Funciones sin propietario claro (distributed ownership)
CREATE VIEW IF NOT EXISTS distributed_ownership AS
SELECT ghf.file_path, ghf.function_name, COUNT(gtc.author) as num_authors
FROM git_function_history ghf
LEFT JOIN git_team_context gtc ON ghf.id = gtc.function_id
WHERE gtc.expertise_level IN ('familiar', 'expert')
GROUP BY ghf.id
HAVING COUNT(gtc.author) > 2
ORDER BY num_authors DESC;

-- Expertise map por autor
CREATE VIEW IF NOT EXISTS author_expertise AS
SELECT
    author,
    COUNT(*) as known_modules,
    SUM(CASE WHEN expertise_level = 'expert' THEN 1 ELSE 0 END) as expert_in,
    SUM(CASE WHEN expertise_level = 'familiar' THEN 1 ELSE 0 END) as familiar_with
FROM git_team_context
GROUP BY author
ORDER BY expert_in DESC;
