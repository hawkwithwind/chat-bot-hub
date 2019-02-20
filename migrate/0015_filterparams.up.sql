CREATE TABLE `filterparams` (
`filterparamid` VARCHAR(36) NOT NULL,
`filtertemplateid` VARCHAR(36) NOT NULL,
`filterparamname` VARCHAR(128) NOT NULL,
`index` INT NOT NULL,
`valueid` VARCHAR(36) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`filterparamid`),
UNIQUE (`valueid`),
INDEX `index_index` (`index`),
INDEX `valueid_index` (`valueid`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
