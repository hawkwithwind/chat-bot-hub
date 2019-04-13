ALTER TABLE `bots` DROP INDEX `accountid_botname_deleteat`;
ALTER TABLE `bots` ADD CONSTRAINT `accountid` UNIQUE (`accountid`, `botname`);

ALTER TABLE `bots` DROP INDEX `login_deleteat`;
ALTER TABLE `bots` ADD CONSTRAINT `login_unique` UNIQUE (`login`);

