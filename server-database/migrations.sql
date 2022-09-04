DROP TABLE IF EXISTS images;

CREATE TABLE images
(
    id serial NOT NULL,
    title text NOT NULL,
    url text NOT NULL,
    alt_text text,
    PRIMARY KEY (id)
);

INSERT INTO images (title, url, alt_text) VALUES ('A cute kitten', 'https://placekitten.com/200/300', 'A kitten looking mischievous');
INSERT INTO images (title, url, alt_text) VALUES ('A cute puppy', 'https://placedog.net/200/300', 'A puppy looking mischievous');