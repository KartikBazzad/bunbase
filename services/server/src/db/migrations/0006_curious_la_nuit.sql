CREATE TYPE "public"."function_status" AS ENUM('draft', 'deployed', 'paused');--> statement-breakpoint
CREATE TABLE "functionDeployment" (
	"id" text PRIMARY KEY NOT NULL,
	"functionId" text NOT NULL,
	"versionId" text NOT NULL,
	"environment" text DEFAULT 'production' NOT NULL,
	"status" text DEFAULT 'active' NOT NULL,
	"deployedAt" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "functionEnvironment" (
	"id" text PRIMARY KEY NOT NULL,
	"functionId" text NOT NULL,
	"key" text NOT NULL,
	"value" text NOT NULL,
	"isSecret" boolean DEFAULT false NOT NULL,
	"createdAt" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "functionEnvironment_functionId_key_unique" UNIQUE("functionId","key")
);
--> statement-breakpoint
CREATE TABLE "functionLog" (
	"id" text PRIMARY KEY NOT NULL,
	"functionId" text NOT NULL,
	"executionId" text NOT NULL,
	"level" text NOT NULL,
	"message" text NOT NULL,
	"metadata" jsonb,
	"timestamp" timestamp DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE "functionMetric" (
	"id" text PRIMARY KEY NOT NULL,
	"functionId" text NOT NULL,
	"date" timestamp NOT NULL,
	"invocations" integer DEFAULT 0 NOT NULL,
	"errors" integer DEFAULT 0 NOT NULL,
	"totalDuration" integer DEFAULT 0 NOT NULL,
	"coldStarts" integer DEFAULT 0 NOT NULL,
	CONSTRAINT "functionMetric_functionId_date_unique" UNIQUE("functionId","date")
);
--> statement-breakpoint
CREATE TABLE "functionVersion" (
	"id" text PRIMARY KEY NOT NULL,
	"functionId" text NOT NULL,
	"version" text NOT NULL,
	"codeHash" text NOT NULL,
	"codePath" text NOT NULL,
	"deployedAt" timestamp,
	"createdAt" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "functionVersion_functionId_version_unique" UNIQUE("functionId","version")
);
--> statement-breakpoint
CREATE TABLE "function" (
	"id" text PRIMARY KEY NOT NULL,
	"projectId" text NOT NULL,
	"name" text NOT NULL,
	"runtime" text DEFAULT 'bun' NOT NULL,
	"handler" text NOT NULL,
	"status" "function_status" DEFAULT 'draft' NOT NULL,
	"memory" integer DEFAULT 512,
	"timeout" integer DEFAULT 30,
	"createdAt" timestamp DEFAULT now() NOT NULL,
	"updatedAt" timestamp DEFAULT now() NOT NULL,
	CONSTRAINT "function_projectId_name_unique" UNIQUE("projectId","name")
);
--> statement-breakpoint
ALTER TABLE "functionDeployment" ADD CONSTRAINT "functionDeployment_functionId_function_id_fk" FOREIGN KEY ("functionId") REFERENCES "public"."function"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "functionDeployment" ADD CONSTRAINT "functionDeployment_versionId_functionVersion_id_fk" FOREIGN KEY ("versionId") REFERENCES "public"."functionVersion"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "functionEnvironment" ADD CONSTRAINT "functionEnvironment_functionId_function_id_fk" FOREIGN KEY ("functionId") REFERENCES "public"."function"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "functionLog" ADD CONSTRAINT "functionLog_functionId_function_id_fk" FOREIGN KEY ("functionId") REFERENCES "public"."function"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "functionMetric" ADD CONSTRAINT "functionMetric_functionId_function_id_fk" FOREIGN KEY ("functionId") REFERENCES "public"."function"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "functionVersion" ADD CONSTRAINT "functionVersion_functionId_function_id_fk" FOREIGN KEY ("functionId") REFERENCES "public"."function"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "function" ADD CONSTRAINT "function_projectId_project_id_fk" FOREIGN KEY ("projectId") REFERENCES "public"."project"("id") ON DELETE cascade ON UPDATE no action;