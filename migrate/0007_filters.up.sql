CREATE TABLE `filters`(
`filterid` VARCHAR(36) NOT NULL,
`filtertemplateid` VARCHAR(36) NOT NULL,
`accountid` VARCHAR(36) NOT NULL,
`filtername` VARCHAR(128) NOT NULL,
`body` TEXT,
`next` VARCHAR(36),
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`filterid`),
UNIQUE KEY (`accountid`, `filtername`),
INDEX `accountid_index` (`accountid`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
