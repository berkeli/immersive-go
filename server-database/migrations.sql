DROP TABLE IF EXISTS images;

CREATE TABLE images
(
    id serial NOT NULL,
    title text NOT NULL,
    url text NOT NULL UNIQUE CONSTRAINT url_is_valid CHECK (url ~* '^(https?|ftp)://[^\s/$.?#].[^\s]*$'),
    width integer NOT NULL,
    height integer NOT NULL,
    alt_text text,
    PRIMARY KEY (id)
);