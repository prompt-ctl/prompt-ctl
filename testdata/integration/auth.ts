import jwt from 'jsonwebtoken';

const SECRET = process.env.JWT_SECRET || 'default-secret';

export interface User {
  id: string;
  email: string;
  role: 'admin' | 'user';
}

export function generateToken(user: User): string {
  return jwt.sign({ sub: user.id, email: user.email, role: user.role }, SECRET, {
    expiresIn: '24h',
  });
}

export function verifyToken(token: string): User | null {
  try {
    const decoded = jwt.verify(token, SECRET) as any;
    return { id: decoded.sub, email: decoded.email, role: decoded.role };
  } catch {
    return null;
  }
}

export function requireAdmin(user: User): boolean {
  return user.role === 'admin';
}
