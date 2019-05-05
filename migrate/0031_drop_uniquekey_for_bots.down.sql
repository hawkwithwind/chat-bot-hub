ALTER TABLE `bots` DROP INDEX `accountid_index`;
ALTER TABLE `bots` DROP INDEX `botname_index`;
ALTER TABLE `bots` DROP INDEX `login_index`;

ALTER TABLE `bots` ADD CONSTRAINT `accountid_botname_deleteat` UNIQUE (`accountid`, `botname`, `deleteat`);
ALTER TABLE `bots` ADD CONSTRAINT `login_deleteat` UNIQUE (`login`, `deleteat`);

