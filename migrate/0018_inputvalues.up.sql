CREATE TABLE `inputvalues` (
`inputvalueid` VARCHAR(36) NOT NULL,
`valueid` VARCHAR(64) NOT NULL,
`type` VARCHAR(64) NOT NULL,
`value` TEXT,
`defaultvalue` TEXT,
`required` INT NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`inputvalueid`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
