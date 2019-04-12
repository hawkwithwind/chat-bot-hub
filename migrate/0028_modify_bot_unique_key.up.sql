ALTER TABLE `bots` DROP INDEX `accountid`;
ALTER TABLE `bots` ADD CONSTRAINT `accountid_botname_deleteat` UNIQUE (`accountid`, `botname`, `deleteat`);

ALTER TABLE `bots` DROP INDEX `login_unique`;
ALTER TABLE `bots` ADD CONSTRAINT `login_deleteat` UNIQUE (`login`, `deleteat`);
