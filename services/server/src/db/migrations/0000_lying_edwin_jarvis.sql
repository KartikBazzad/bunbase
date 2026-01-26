CREATE TABLE `applicationApiKey` (
	`id` text PRIMARY KEY NOT NULL,
	`applicationId` text NOT NULL,
	`keyHash` text NOT NULL,
	`keyPrefix` text NOT NULL,
	`keySuffix` text NOT NULL,
	`createdAt` integer NOT NULL,
	`lastUsedAt` integer,
	`revokedAt` integer,
	FOREIGN KEY (`applicationId`) REFERENCES `application`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `application` (
	`id` text PRIMARY KEY NOT NULL,
	`projectId` text NOT NULL,
	`name` text NOT NULL,
	`description` text NOT NULL,
	`type` text DEFAULT 'web' NOT NULL,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`projectId`) REFERENCES `project`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `collection` (
	`collectionId` text PRIMARY KEY NOT NULL,
	`databaseId` text NOT NULL,
	`name` text NOT NULL,
	`path` text NOT NULL,
	`parentDocumentId` text,
	`parentPath` text,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`databaseId`) REFERENCES `database`(`databaseId`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE UNIQUE INDEX `collection_path_unique` ON `collection` (`path`);--> statement-breakpoint
CREATE TABLE `database` (
	`databaseId` text PRIMARY KEY NOT NULL,
	`name` text NOT NULL,
	`projectId` text NOT NULL,
	`isDefault` integer DEFAULT false NOT NULL,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`projectId`) REFERENCES `project`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `functionDeployment` (
	`id` text PRIMARY KEY NOT NULL,
	`functionId` text NOT NULL,
	`versionId` text NOT NULL,
	`environment` text DEFAULT 'production' NOT NULL,
	`status` text DEFAULT 'active' NOT NULL,
	`deployedAt` integer NOT NULL,
	FOREIGN KEY (`functionId`) REFERENCES `function`(`id`) ON UPDATE no action ON DELETE cascade,
	FOREIGN KEY (`versionId`) REFERENCES `functionVersion`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `functionEnvironment` (
	`id` text PRIMARY KEY NOT NULL,
	`functionId` text NOT NULL,
	`key` text NOT NULL,
	`value` text NOT NULL,
	`isSecret` integer DEFAULT false NOT NULL,
	`createdAt` integer NOT NULL,
	FOREIGN KEY (`functionId`) REFERENCES `function`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `functionLog` (
	`id` text PRIMARY KEY NOT NULL,
	`functionId` text NOT NULL,
	`executionId` text NOT NULL,
	`level` text NOT NULL,
	`message` text NOT NULL,
	`metadata` text,
	`timestamp` integer NOT NULL,
	FOREIGN KEY (`functionId`) REFERENCES `function`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `functionMetric` (
	`id` text PRIMARY KEY NOT NULL,
	`functionId` text NOT NULL,
	`date` integer NOT NULL,
	`invocations` integer DEFAULT 0 NOT NULL,
	`errors` integer DEFAULT 0 NOT NULL,
	`totalDuration` integer DEFAULT 0 NOT NULL,
	`coldStarts` integer DEFAULT 0 NOT NULL,
	FOREIGN KEY (`functionId`) REFERENCES `function`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `functionMetricMinute` (
	`id` text PRIMARY KEY NOT NULL,
	`functionId` text NOT NULL,
	`timestamp` integer NOT NULL,
	`invocations` integer DEFAULT 0 NOT NULL,
	`errors` integer DEFAULT 0 NOT NULL,
	`totalDuration` integer DEFAULT 0 NOT NULL,
	`coldStarts` integer DEFAULT 0 NOT NULL,
	FOREIGN KEY (`functionId`) REFERENCES `function`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `functionVersion` (
	`id` text PRIMARY KEY NOT NULL,
	`functionId` text NOT NULL,
	`version` text NOT NULL,
	`codeHash` text NOT NULL,
	`codePath` text NOT NULL,
	`deployedAt` integer,
	`createdAt` integer NOT NULL,
	FOREIGN KEY (`functionId`) REFERENCES `function`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `function` (
	`id` text PRIMARY KEY NOT NULL,
	`projectId` text NOT NULL,
	`name` text NOT NULL,
	`runtime` text DEFAULT 'bun' NOT NULL,
	`handler` text NOT NULL,
	`status` text DEFAULT 'draft' NOT NULL,
	`memory` integer DEFAULT 512,
	`timeout` integer DEFAULT 30,
	`maxConcurrentExecutions` integer DEFAULT 10,
	`runtimeType` text DEFAULT 'worker',
	`activeVersionId` text,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`projectId`) REFERENCES `project`(`id`) ON UPDATE no action ON DELETE cascade,
	FOREIGN KEY (`activeVersionId`) REFERENCES `functionVersion`(`id`) ON UPDATE no action ON DELETE set null
);
--> statement-breakpoint
CREATE TABLE `projectAuth` (
	`projectId` text PRIMARY KEY NOT NULL,
	`providers` text DEFAULT '["email"]' NOT NULL,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`projectId`) REFERENCES `project`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `projectAuthSettings` (
	`projectId` text PRIMARY KEY NOT NULL,
	`requireEmailVerification` integer DEFAULT false NOT NULL,
	`rateLimitMax` integer DEFAULT 5 NOT NULL,
	`rateLimitWindow` integer DEFAULT 15 NOT NULL,
	`sessionExpirationDays` integer DEFAULT 30 NOT NULL,
	`minPasswordLength` integer DEFAULT 8 NOT NULL,
	`requireUppercase` integer DEFAULT false NOT NULL,
	`requireLowercase` integer DEFAULT false NOT NULL,
	`requireNumbers` integer DEFAULT false NOT NULL,
	`requireSpecialChars` integer DEFAULT false NOT NULL,
	`mfaEnabled` integer DEFAULT false NOT NULL,
	`mfaRequired` integer DEFAULT false NOT NULL,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`projectId`) REFERENCES `project`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `projectOAuthProvider` (
	`id` text PRIMARY KEY NOT NULL,
	`projectId` text NOT NULL,
	`provider` text NOT NULL,
	`clientId` text NOT NULL,
	`clientSecret` text NOT NULL,
	`redirectUri` text,
	`scopes` text DEFAULT '[]',
	`isConfigured` integer DEFAULT false NOT NULL,
	`lastTestedAt` integer,
	`lastTestStatus` text,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`projectId`) REFERENCES `project`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `project` (
	`id` text PRIMARY KEY NOT NULL,
	`name` text NOT NULL,
	`description` text NOT NULL,
	`ownerId` text NOT NULL,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`ownerId`) REFERENCES `user`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `session` (
	`id` text PRIMARY KEY NOT NULL,
	`userId` text NOT NULL,
	`token` text NOT NULL,
	`expiresAt` integer NOT NULL,
	`ipAddress` text,
	`userAgent` text,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`userId`) REFERENCES `user`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE UNIQUE INDEX `session_token_unique` ON `session` (`token`);--> statement-breakpoint
CREATE TABLE `storage` (
	`storageId` text PRIMARY KEY NOT NULL,
	`name` text NOT NULL,
	`projectId` text NOT NULL,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`projectId`) REFERENCES `project`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `userAccount` (
	`id` text PRIMARY KEY NOT NULL,
	`userId` text NOT NULL,
	`accountId` text NOT NULL,
	`providerId` text NOT NULL,
	`accessToken` text,
	`refreshToken` text,
	`accessTokenExpiresAt` integer,
	`refreshTokenExpiresAt` integer,
	`scope` text,
	`idToken` text,
	`password` text,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL,
	FOREIGN KEY (`userId`) REFERENCES `user`(`id`) ON UPDATE no action ON DELETE cascade
);
--> statement-breakpoint
CREATE TABLE `user` (
	`id` text PRIMARY KEY NOT NULL,
	`name` text NOT NULL,
	`email` text NOT NULL,
	`emailVerified` integer DEFAULT false NOT NULL,
	`image` text,
	`isBanned` integer DEFAULT false NOT NULL,
	`banReason` text,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL
);
--> statement-breakpoint
CREATE UNIQUE INDEX `user_email_unique` ON `user` (`email`);--> statement-breakpoint
CREATE TABLE `verification` (
	`id` text PRIMARY KEY NOT NULL,
	`identifier` text NOT NULL,
	`value` text NOT NULL,
	`expiresAt` integer NOT NULL,
	`createdAt` integer NOT NULL,
	`updatedAt` integer NOT NULL
);
