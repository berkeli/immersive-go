How I approached this problem:

* Ensure the app is running locally:
   * `make build run`
   * `make migrate`
   * `go run ./cmd/test user -password banana`
   * `curl 127.0.0.1:8090/1/my/notes.json -H 'Authorization: Basic X0VKaXkyUEo6YmFuYW5h' -i`
These commands helped me verify that the app is running as expected and I've received and empty array of my notes.

* Reading through the requirements, there are at least 5 or more bugs. There are issues in the logic, performance, and authorisation (that we know about). There could be more! Find and fix as many as you can in the time.

* There are 2 tables in the database users and notes. Notes have tags which are not stored in db and extracted from the content at runtime.

* As I'm going through the notes there are 2 requirements:
  1. Users with status inactive should not be able to authenticate or access their notes.
  2. Users should not be able to access notes that they do not own.

These 2 requirements already give me a few business rules that I test against. So I have connected to postgres database via CLI and will be doing some manual tests:

#### Bug number 1
1. I have added a note for inactive user and checked if I can access notes of that user, and found the first bug:
    - ID of inactive user is `noJi6S_V`
    - Let's create a note for inactive user `go run ./cmd/test note -owner noJi6S_V`
    - Check notes of user `noJi6S_V` via curl shows us:
```
{
    "notes": [
        {
            "id": "VbkYyftX",
            "owner": "noJi6S_V",
            "content": "Example note content",
            "created": "2022-11-08T16:00:32.405064Z",
            "modified": "2022-11-08T16:00:32.405064Z",
            "tags": []
        }
    ]
}
```
I decided to resolve this issue via TDD, as it is easier to follow TDD principles in an already established project. For me it's always difficult to test something that I don't yet know the architecture for.

I checked the [auth_test.go](auth/auth_test.go) and will be doing some refactoring to implement table tests for verification. After refactoring and writing table tests, I received the following result:
```
--- FAIL: TestVerify (0.93s)
    --- PASS: TestVerify/valid_user_and_password (0.29s)
    --- PASS: TestVerify/valid_user_with_invalid_password (0.22s)
    --- FAIL: TestVerify/inactive_user_with_valid_password (0.21s)
    --- PASS: TestVerify/inactive_user_with_invalid_password (0.22s)
```
Based on this I need to modify auth service so `TestVerify/inactive_user_with_valid_password` test case passes. The modification I made is as follows:
before:
```
	err := as.pool.QueryRow(ctx,
		"SELECT id, password, status FROM public.user WHERE id = $1",
		in.Id,
	).Scan(&row.id, &row.password, &row.status)
```
after:
```
	err := as.pool.QueryRow(ctx,
		"SELECT id, password, status FROM public.user WHERE id = $1 AND status = $2",
		in.Id,
		"active",
	).Scan(&row.id, &row.password, &row.status)
```
I'm essentially filtering out users that are not active in the query to the database. I'm explicitly filtering users that have a status of active, as doing a filter such as `status != inactive` might lead to users with a typo in status or not status gaining access to the system.

#### Bug number 2
After fixing this bug I wanted to test the 2nd explicit requirement from the readme: Users should not be able to access notes that they do not own.

To test, I checked for all notes in the DB via terminal (`SELECT * FROM public.note`) and got the id for note and userId. Then I tried to access the note with credentials of a different user.

```
postgres@localhost:app> select * from public.note;
+----------+----------+----------------------+----------------------------+----------------------------+
| id       | owner    | content              | created                    | modified                   |
|----------+----------+----------------------+----------------------------+----------------------------|
| O-VN1g7v | _EJiy2PJ | example #tag1 #tag33 | 2022-11-08 15:54:44.183083 | 2022-11-08 15:57:13.833892 |
| VbkYyftX | noJi6S_V | Example note content | 2022-11-08 16:00:32.405064 | 2022-11-08 16:00:32.405064 |
+----------+----------+----------------------+----------------------------+----------------------------+
```
I checked the note id `O-VN1g7v` with user `noJi6S_V`'s credentials and it was not rejecting me.
```
❯ curl 127.0.0.1:8090/1/my/note/O-VN1g7v.json \
        -H 'Authorization: Basic X0VKaXkyUEo6YmFuYW5h' -i
HTTP/1.1 200 OK
Content-Type: text/json
Date: Tue, 08 Nov 2022 17:42:06 GMT
Content-Length: 183

{"note":{"id":"O-VN1g7v","owner":"_EJiy2PJ","content":"example #tag1 #tag33","created":"2022-11-08T15:54:44.183083Z","modified":"2022-11-08T15:57:13.833892Z","tags":["tag1","tag33"]}}%
```

To fix the issue, I will follow the same TDD methodology. After reviewing the setup I realized there are 2 *main* ways of implementing this.
1. Get the note and check if owner of that note is authenticated user, reject the request if it's not. 
2. In the [GetNodeById](api/model/notes.go) function, modify DB query so it looks for notes by ID and Owner.

There are pros and cons to both solutions, but I decided to go with the 1st approach as it clearly sends a message to the user that they do not have access to it. If we send note not found, it might confuse the user since they are making a request by ID.

After modifying the [api_test.go](api/api_test.go) I got the following test result to fix:
```
--- FAIL: TestMyNoteById (0.00s)
    --- PASS: TestMyNoteById/note_belongs_to_user (0.00s)
    --- FAIL: TestMyNoteById/note_belongs_to_different_user (0.00s)
```







