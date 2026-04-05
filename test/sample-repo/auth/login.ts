import { getUser } from '../db/user';
import { hashPassword, validateEmail, generateToken } from '../utils/helpers';

/**
 * Authenticates a user with email and password.
 * Returns a session token on success.
 */
export async function login(email: string, password: string): Promise<string | null> {
  if (!validateEmail(email)) {
    throw new Error('Invalid email format');
  }

  const user = await getUser(email);
  if (!user) {
    return null;
  }

  const hashedPassword = hashPassword(password);
  if (user.password !== hashedPassword) {
    return null;
  }

  const token = generateToken();
  await createSession(user.id, token);
  return token;
}

/**
 * Logs out a user by invalidating their session.
 */
export async function logout(token: string): Promise<void> {
  await deleteSession(token);
}

/**
 * Creates a new user session.
 */
async function createSession(userId: string, token: string): Promise<void> {
  // Store session in database
}

/**
 * Deletes a user session.
 */
async function deleteSession(token: string): Promise<void> {
  // Remove session from database
}
