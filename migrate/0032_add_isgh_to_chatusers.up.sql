ALTER TABLE `chatusers` ADD `isgh` INT DEFAULT 0;
ALTER TABLE `chatusers` ADD INDEX `isgh_index` (`isgh`);
