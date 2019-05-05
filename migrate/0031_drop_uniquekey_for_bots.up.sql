ALTER TABLE `bots` DROP INDEX `accountid_botname_deleteat`;
ALTER TABLE `bots` DROP INDEX `login_deleteat`;

ALTER TABLE `bots` ADD INDEX `accountid_index` (`accountid`);
ALTER TABLE `bots` ADD INDEX `botname_index` (`botname`);
ALTER TABLE `bots` ADD INDEX `login_index` (`login`);
