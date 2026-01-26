-- Add new columns to projectAuthSettings
ALTER TABLE "projectAuthSettings" ADD COLUMN "minPasswordLength" integer DEFAULT 8 NOT NULL;--> statement-breakpoint
ALTER TABLE "projectAuthSettings" ADD COLUMN "requireUppercase" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "projectAuthSettings" ADD COLUMN "requireLowercase" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "projectAuthSettings" ADD COLUMN "requireNumbers" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "projectAuthSettings" ADD COLUMN "requireSpecialChars" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "projectAuthSettings" ADD COLUMN "mfaEnabled" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "projectAuthSettings" ADD COLUMN "mfaRequired" boolean DEFAULT false NOT NULL;--> statement-breakpoint
-- Create projectOAuthProvider table
CREATE TABLE "projectOAuthProvider" (
	"id" text PRIMARY KEY NOT NULL,
	"projectId" text NOT NULL,
	"provider" "auth_provider" NOT NULL,
	"clientId" text NOT NULL,
	"clientSecret" text NOT NULL,
	"redirectUri" text,
	"scopes" jsonb DEFAULT '[]'::jsonb,
	"isConfigured" boolean DEFAULT false NOT NULL,
	"lastTestedAt" timestamp,
	"lastTestStatus" text,
	"createdAt" timestamp DEFAULT now() NOT NULL,
	"updatedAt" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
ALTER TABLE "projectOAuthProvider" ADD CONSTRAINT "projectOAuthProvider_projectId_project_id_fk" FOREIGN KEY ("projectId") REFERENCES "public"."project"("id") ON DELETE cascade ON UPDATE no action;
