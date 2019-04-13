ALTER TABLE `filters` DROP INDEX `accountid`;
ALTER TABLE `filters` ADD CONSTRAINT `accountid_filtername_deleteat` UNIQUE (`accountid`, `filtername`, `deleteat`);
