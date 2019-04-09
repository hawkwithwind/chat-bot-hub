ALTER TABLE `bots` ADD `momentfilterid` VARCHAR(36);
ALTER TABLE `bots` ADD INDEX `momentfilterid_index` (`momentfilterid`);
