# GoAggreate

Aggregates 2 and more apis with a configurable yaml file.

![Combine 2 API endpoints](http://www.plantuml.com/plantuml/png/XP0nItDH38Rt_8fmftRuFj3HGIeYBkAWZYwX9_Vss7EIqoH7wR-tfHGhXRebF3nloBlm88jU2qIEmlYeTzDaJC40lixIoAXYLT7bohGoXdK-8RwYf5zP9XofE0FGCjHmk2-P9GQ0uuJ_Rd7uIHzky8KtdJtsHlwau5yOu7GxbvzKrxtXKcNCKpUAhF8kiyF-VrlB7A2eDdnbIMY7KuwUQoz1mjgzR07nxCg3fAPP0g-6Y_ZwCduUW2NzzSMRaRYELj7OcugpxvwWrieMFpOuvX9iy_EKOwdx4WyUpcAVDQfLJfWeQASjM5AlvHi0)

## Why I need this?

If you want to get programming quotes from [Programming Quotes API](https://github.com/skolakoda/programming-quotes-api) and programming jokes from [JokeAPI](https://sv443.net/jokeapi/v2) but you don't want to send multipe requests and transform data in frontend instead of backend. GoAggreate is here for you.

## Example

### Combine 2 API endpoints

Here is what original api looks like:

1. Programming Quote API

    **GET** https://sv443.net/jokeapi/v2/joke/Programming?type=single

    ```javascript
    {
        "_id": "5a6ce8702af929789500e8cc",
        "sr": "Matematičari stoje jedni drugima na ramenima, a kompjuterski naučnici stoje jedni drugima na prstima.",
        "en": "Mathematicians stand on each others' shoulders and computer scientists stand on each others' toes.",
        "author": "Richard Hamming",
        "source": "",
        "numberOfVotes": 3,
        "rating": 3.3,
        "addedBy": "5ab04d928c8b4e3cbf733557",
        "id": "5a6ce8702af929789500e8cc"
    }
    ```

2. JokeAPI

    **GET** https://programming-quotes-api.herokuapp.com/quotes/random/lang/en

    ```javascript
    {
        "category": "Programming",
        "type": "single",
        "joke": "A byte walks into a bar looking miserable.\nThe bartender asks him: \"What's wrong buddy?\"\n\"Parity error.\" he replies. \n\"Ah that makes sense, I thought you looked a bit off.\"",
        "flags": {
            "nsfw": false,
            "religious": false,
            "political": false,
            "racist": false,
            "sexist": false
        },
        "id": 24,
        "error": false
    }
    ```

#### Create YAML Configuration file \"config.yml\"

```yaml
apis:
programming-jokeandquote:
    name: programming-jokeandquote
    jq: '.'
    endpoints:
    - name: joke
    endpoint: https://sv443.net/jokeapi/v2/joke/Programming?type=single
    ssl_verify: true
    headers:
        extraHeaders: 'value'
    - name: quote
    endpoint: https://programming-quotes-api.herokuapp.com/quotes/random/lang/en
    ssl_verify: true
    headers:
        extraHeaders: 'value'

```

Strt GoAggreate `./goaggreate`, the server will start listening on port 8080 by default. (Use env `PORT` to change listen port).

**GET** http://localhost/programming-jokeandquote

Response

```javascript
{
    "quote": {
        "_id": "5a6ce8702af929789500e8cc",
        "sr": "Matematičari stoje jedni drugima na ramenima, a kompjuterski naučnici stoje jedni drugima na prstima.",
        "en": "Mathematicians stand on each others' shoulders and computer scientists stand on each others' toes.",
        "author": "Richard Hamming",
        "source": "",
        "numberOfVotes": 3,
        "rating": 3.3,
        "addedBy": "5ab04d928c8b4e3cbf733557",
        "id": "5a6ce8702af929789500e8cc"
    },
    "joke": {
        "category": "Programming",
        "type": "single",
        "joke": "A byte walks into a bar looking miserable.\nThe bartender asks him: \"What's wrong buddy?\"\n\"Parity error.\" he replies. \n\"Ah that makes sense, I thought you looked a bit off.\"",
        "flags": {
            "nsfw": false,
            "religious": false,
            "political": false,
            "racist": false,
            "sexist": false
        },
        "id": 24,
        "error": false
    }
}
```

#### Transform data with JQ filter

If you don't just want to merge endpoints, try to use `jq` to process the data with your custom filter.

Update `config.yml` jq field from `.` to `add`. Now the response will be

```javascript
{
    "_id": "5aad71337632ba0004ec84b2",
    "en": "The more varieties of different kinds of notations are still useful — don’t only read the people who code like you.",
    "sr": "",
    "author": "Donald Knuth",
    "source": "Coders at Work",
    "numberOfVotes": 2,
    "rating": 4.3,
    "__v": 0,
    "addedBy": "5ab04d928c8b4e3cbf733557",
    "id": 27,
    "category": "Programming",
    "type": "single",
    "joke": "Java is like Alzheimer, it starts off slow, but eventually, your memory is gone.",
    "flags": {
        "nsfw": false,
        "religious": false,
        "political": false,
        "racist": false,
        "sexist": false
    },
    "error": false
}
```

You may not want all the fields in the endpoints. Try to modify jq field from `.` to `{"quote": (.quote.en), "joke": (.joke.joke)}`

The output will be

```javascript
{
  "quote": "The cost of adding a feature isn’t just the time it takes to code it. The cost also includes the addition of an obstacle to future expansion. The trick is to pick the features that don’t fight each other.",
  "joke": "Judge: \"I sentence you to the maximum punishment...\"\nMe (thinking): \"Please be death, please be death...\"\nJudge: \"Learn Java!\"\nMe: \"Damn.\""
}
```
