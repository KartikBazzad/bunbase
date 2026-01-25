CREATE TABLE "applicationApiKey" (
	"id" text PRIMARY KEY NOT NULL,
	"applicationId" text NOT NULL,
	"keyHash" text NOT NULL,
	"keyPrefix" text NOT NULL,
	"keySuffix" text NOT NULL,
	"createdAt" timestamp DEFAULT now() NOT NULL,
	"lastUsedAt" timestamp,
	"revokedAt" timestamp
);
--> statement-breakpoint
ALTER TABLE "applicationApiKey" ADD CONSTRAINT "applicationApiKey_applicationId_application_id_fk" FOREIGN KEY ("applicationId") REFERENCES "public"."application"("id") ON DELETE cascade ON UPDATE no action;