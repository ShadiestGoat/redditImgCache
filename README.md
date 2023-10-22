# Reddit Cache Server

This is a server for caching subreddits image posts. 

## Features

- Cache image posts
- Cache if an image is NSFW
- Cache the subreddit the image is in
- Alias subreddits - can be used to merge 2 subreddits together
- Cache if an image is nsfw
- Query random images from a subreddit

## Config

The config is stored in a `config.yaml` file. Check `config.example.yaml` for defaults & documentation.

## How to use

1. `go install github.com/shadiestgoat/redditImgCache@latest` ([set up go if needed](https://go.dev/doc/install))
2. Configure the `config.yaml` (see above section)
3. Run the `redditImgCache` command in the same directory as the config file

Now you can do:

### GET /subs

Returns a map of aliased subreddit name -> n of posts cached
Also could return a `{"error": "DB Error"}`. Create an issue in this case - this shouldn't happen, but theoretically could.

### GET /r/{sub}

Where `{sub}` is an aliased name of a subreddit. You can include a query parameter `nsfw`. This can be 0, 1 or -1:

| Value | Explanation |
|:-----:|:------------|
| 0 | Does nothing. No nsfw filter. |
| -1 | Do not return an nsfw post. |
| 1 | Must return an nsfw post. |

Returns:
```json
{
    "img": string,
    "nsfw": bool,
    "width": int,
    "height": int,
}
```

## How it works

This project uses the reddit json api. Its not really supported, or documented, or anything like that, so fair warning about this breaking.

The idea is that this project will cache all the posts it can in an internal database rather than relying on the paginated results that reddit provides. So heres what happens:

1. At startup, it checks to see what needs to be fetched. Any new subreddits will fetch using the top endpoint, going from top of all time -> worst of all time. If the subreddit already has a cache, it does the same as during a poll/hydration
2. Each subreddit creates a periodic job, that runs every `hydrate` hours (see config). This job will fetch all the posts that come before the latest cached id

Just as a note - because posts are cached once & never revisited, if a post isn't right for a sub reddit (yet to be moderated) it wouldn't be removed. Thats why a post needs to be up for at least 4 hours before it gets added to the cache.