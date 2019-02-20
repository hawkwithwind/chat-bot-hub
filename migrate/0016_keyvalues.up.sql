CREATE TABLE `keyvalues` (
`keyvalueid` VARCHAR(36) NOT NULL,
`valueid` VARCHAR(36) NOT NULL,
`keytype` VARCHAR(64) NOT NULL,
`key` VARCHAR(128) NOT NULL,
`valuetype` VARCHAR(64) NOT NULL,
`value` VARCHAR(128) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`keyvalueid`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
