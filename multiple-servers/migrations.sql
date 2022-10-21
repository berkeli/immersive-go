DROP TABLE IF EXISTS images;

CREATE TABLE images (
  id SERIAL PRIMARY KEY,
  title text NOT NULL,
  alt_text text,
  url text NOT NULL
);

INSERT INTO images (title, alt_text, url) VALUES ('A cute kitten', 'A kitten looking mischievous', 'https://placekitten.com/200/300');
INSERT INTO images (title, alt_text, url) VALUES ('A cute puppy', 'A puppy looking mischievous', 'https://placedog.net/200/300');