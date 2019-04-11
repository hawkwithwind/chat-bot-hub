ALTER TABLE `filters` DROP INDEX `accountid_filtername_deleteat`;
ALTER TABLE `filters` ADD INDEX `accountid` (`accountid`, `filtername`);

