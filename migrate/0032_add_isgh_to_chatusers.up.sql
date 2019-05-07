ALTER TABLE `chatusers` ADD `isgh` INT DEFAULT 0;
ALTER TABLE `chatusers` ADD INDEX `isgh_index` (`isgh`);

UPDATE `chatusers` SET isgh=1 WHERE `username` LIKE 'gh_%';
