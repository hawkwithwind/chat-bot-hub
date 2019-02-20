CREATE TABLE `filtergeneratorinputs` (
`filtergeneratorinputid` VARCHAR(36) NOT NULL,
`filtergeneratorid` VARCHAR(36) NOT NULL,
`valueid` VARCHAR(36) NOT NULL,
`inputvalueid` VARCHAR(36) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`filtergeneratorinputid`),
UNIQUE (`valueid`),
UNIQUE (`inputvalueid`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
