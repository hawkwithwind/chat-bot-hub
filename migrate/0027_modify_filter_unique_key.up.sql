ALTER TABLE `filters` DROP INDEX `accountid`;
ALTER TABLE `filters` ADD INDEX `accountid_filtername_deleteat` (`accountid`, `filtername`, `deleteat`);
