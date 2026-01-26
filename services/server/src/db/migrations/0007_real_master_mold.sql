CREATE TABLE "functionMetricMinute" (
	"id" text PRIMARY KEY NOT NULL,
	"functionId" text NOT NULL,
	"timestamp" timestamp NOT NULL,
	"invocations" integer DEFAULT 0 NOT NULL,
	"errors" integer DEFAULT 0 NOT NULL,
	"totalDuration" integer DEFAULT 0 NOT NULL,
	"coldStarts" integer DEFAULT 0 NOT NULL,
	CONSTRAINT "functionMetricMinute_functionId_timestamp_unique" UNIQUE("functionId","timestamp")
);
--> statement-breakpoint
ALTER TABLE "function" ADD COLUMN "maxConcurrentExecutions" integer DEFAULT 10;--> statement-breakpoint
ALTER TABLE "function" ADD COLUMN "runtimeType" text DEFAULT 'worker';--> statement-breakpoint
ALTER TABLE "function" ADD COLUMN "activeVersionId" text;--> statement-breakpoint
ALTER TABLE "functionMetricMinute" ADD CONSTRAINT "functionMetricMinute_functionId_function_id_fk" FOREIGN KEY ("functionId") REFERENCES "public"."function"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "function" ADD CONSTRAINT "function_activeVersionId_functionVersion_id_fk" FOREIGN KEY ("activeVersionId") REFERENCES "public"."functionVersion"("id") ON DELETE set null ON UPDATE no action;