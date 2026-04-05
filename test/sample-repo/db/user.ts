export async function getUser(id: string) {
  return db.query('SELECT * FROM users WHERE id = ?', [id]);
}

export async function updateUser(id: string, data) {
  return db.query('UPDATE users SET ...', [id, data]);
}

export async function deleteUser(id: string) {
  return db.query('DELETE FROM users WHERE id = ?', [id]);
}

export async function listUsers(limit: number = 100) {
  return db.query('SELECT * FROM users LIMIT ?', [limit]);
}
