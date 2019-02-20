CREATE TABLE `filtertemplates` (
`filtertemplateid` VARCHAR(36) NOT NULL,
`accountid` VARCHAR(36) NOT NULL,
`filtertemplatename` VARCHAR(128) NOT NULL,
`filtertemplatesuiteid` VARCHAR(36) NOT NULL,
`index` INT NOT NULL,
`type`  VARCHAR(64) NOT NULL,
`defaultnext` INT NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
`updateat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
`deleteat` DATETIME DEFAULT NULL,
PRIMARY KEY (`filtertemplateid`),
UNIQUE (`accountid`, `filtertemplatename`),
INDEX `index_index` (`index`),
INDEX `type_index` (`type`),
INDEX `createat_index` (`createat`),
INDEX `updateat_index` (`updateat`),
INDEX `deleteat_index` (`deleteat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
