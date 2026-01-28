import { z } from "zod";

const baseUserSchema = z.object({
  id: z.string(),
  name: z.string(),
  email: z.string(),
  emailVerified: z.boolean(),
  image: z.string().nullable(),
  createdAt: z.date(),
  updatedAt: z.date(),
});
export const InternalUserSchema = baseUserSchema;

export type InternalUser = z.infer<typeof InternalUserSchema>;

const baseSessionSchema = z.object({
  id: z.string(),
  userId: z.string(),
  token: z.string(),
  expiresAt: z.date(),
  ipAddress: z.string(),
  userAgent: z.string(),
  createdAt: z.date(),
  updatedAt: z.date(),
});

export const InternalSessionSchema = baseSessionSchema;

export type InternalSession = z.infer<typeof InternalSessionSchema>;

const baseUserAccountSchema = z.object({
  id: z.string(),
  userId: z.string(),
  providerId: z.string(),
  accessToken: z.string().nullable().optional(),
  refreshToken: z.string().nullable().optional(),
  accessTokenExpiresAt: z.date().nullable().optional(),
  refreshTokenExpiresAt: z.date().nullable().optional(),
  scope: z.string().optional().nullable(),
  idToken: z.string().nullable().optional(),
  password: z.string().nullable().optional(), // hashed password
  createdAt: z.date(),
  updatedAt: z.date(),
});

export const InternalUserAccountSchema = baseUserAccountSchema;

export type InternalUserAccount = z.infer<typeof InternalUserAccountSchema>;

const baseVerificationSchema = z.object({
  id: z.string(),
  identifier: z.string(),
  value: z.string(),
  expiresAt: z.date(),
  createdAt: z.date(),
  updatedAt: z.date(),
});

export const InternalVerificationSchema = baseVerificationSchema;

export type InternalVerification = z.infer<typeof InternalVerificationSchema>;

export const UserSchema = baseUserSchema;
export const SessionSchema = baseSessionSchema;
export const UserAccountSchema = baseUserAccountSchema;
export const VerificationSchema = baseVerificationSchema;

export type User = z.infer<typeof UserSchema>;
export type Session = z.infer<typeof SessionSchema>;
export type UserAccount = z.infer<typeof UserAccountSchema>;
export type Verification = z.infer<typeof VerificationSchema>;

export const ProjectSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string(),
  ownerId: z.string(),
  createdAt: z.date(),
  updatedAt: z.date(),
});

export type Project = z.infer<typeof ProjectSchema>;

export const ApplicationSchema = z.object({
  id: z.string(),
  projectId: z.string(),
  name: z.string(),
  description: z.string(),
  type: z.enum(["web"]).default("web"),
  createdAt: z.date(),
  updatedAt: z.date(),
});

export type Application = z.infer<typeof ApplicationSchema>;

export const InternalDatabaseSchema = z.object({
  databaseId: z.string(),
  name: z.string(),
  projectId: z.string(),
  createdAt: z.date(),
  updatedAt: z.date(),
});

export type InternalDatabase = z.infer<typeof InternalDatabaseSchema>;

export const ProjectAuthSchema = z.object({
  projectId: z.string(),
  createdAt: z.date(),
  updatedAt: z.date(),
  providers: z
    .array(z.enum(["email", "google", "github", "facebook", "apple"]))
    .default(["email"]),
});

export type ProjectAuth = z.infer<typeof ProjectAuthSchema>;

export const StorageSchema = z.object({
  storageId: z.string(),
  name: z.string(),
  projectId: z.string(),
  createdAt: z.date(),
  updatedAt: z.date(),
});

export type Storage = z.infer<typeof StorageSchema>;
