ALTER TABLE chats 
ADD COLUMN encryption_algorithm VARCHAR(50),
ADD COLUMN encryption_mode VARCHAR(50),
ADD COLUMN encryption_padding VARCHAR(50); 