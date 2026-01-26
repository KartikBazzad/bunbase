CREATE TABLE "projectAuthSettings" (
	"projectId" text PRIMARY KEY NOT NULL,
	"requireEmailVerification" boolean DEFAULT false NOT NULL,
	"rateLimitMax" integer DEFAULT 5 NOT NULL,
	"rateLimitWindow" integer DEFAULT 15 NOT NULL,
	"sessionExpirationDays" integer DEFAULT 30 NOT NULL,
	"createdAt" timestamp DEFAULT now() NOT NULL,
	"updatedAt" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
ALTER TABLE "user" ADD COLUMN "isBanned" boolean DEFAULT false NOT NULL;--> statement-breakpoint
ALTER TABLE "user" ADD COLUMN "banReason" text;--> statement-breakpoint
ALTER TABLE "projectAuthSettings" ADD CONSTRAINT "projectAuthSettings_projectId_project_id_fk" FOREIGN KEY ("projectId") REFERENCES "public"."project"("id") ON DELETE cascade ON UPDATE no action;