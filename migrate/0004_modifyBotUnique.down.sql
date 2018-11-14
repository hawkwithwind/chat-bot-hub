ALTER TABLE `bots` DROP INDEX `login_unique`;
CREATE INDEX `accountid_2` ON `bots`(`accountid`, `login`);
CREATE INDEX `login_index` ON `bots`(`login`);
