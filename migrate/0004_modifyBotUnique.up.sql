ALTER TABLE `bots` DROP INDEX `login_index`;
ALTER TABLE `bots` DROP INDEX `accountid_2`;
CREATE UNIQUE INDEX `login_unique` ON `bots`(`login`);
