CREATE TABLE `moments` (
`momentid` VARCHAR(36) NOT NULL,
`botid` VARCHAR(36) NOT NULL,
`momentcode` VARCHAR(36) NOT NULL,
`sendat` DATETIME NOT NULL,
`chatuserid` VARCHAR(36) NOT NULL,
`createat` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
PRIMARY KEY (`momentid`),
UNIQUE KEY (`botid`, `momentcode`),
INDEX `chatuserid_index` (`chatuserid`),
INDEX `sendat_index` (`sendat`),
INDEX `createat_index` (`createat`)
)
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
