CREATE TABLE `filtertemplatesuites` (
`filtertemplatesuiteid` VARCHAR(36) NOT NULL,
`accountid` VARCHAR(36) NOT NULL,
`filtertemplatesuitename` VARCHAR(128) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`filtertemplatesuiteid`),
UNIQUE (`accountid`, `filtertemplatesuitename`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
