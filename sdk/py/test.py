import asyncio

from stremthru import StremThru

st = StremThru(base_url="http://localhost:8080", auth="root:root")


async def run():
    res = await st.store.get_user()
    res = await st.store.list_magnets()
    res = await st.store.get_magnet(res.data["items"][0]["id"])
    res = await st.store.generate_link(res.data["files"][0]["link"])
    print(res)


asyncio.run(run())
