import * as crypto from 'crypto';

export function hashPassword(pass: string): string {
  return crypto.createHash('sha256').update(pass).digest('hex');
}

export function validateEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

export function generateToken(length: number = 32): string {
  return crypto.randomBytes(length).toString('hex');
}

export function formatDate(date: Date): string {
  return date.toISOString().split('T')[0];
}
