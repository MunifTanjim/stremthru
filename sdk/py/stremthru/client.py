import base64
import io
from dataclasses import dataclass
from typing import (
    Any,
    Generic,
    Literal,
    Mapping,
    Optional,
    TypedDict,
    TypeVar,
    Union,
    cast,
)

import aiohttp
from multidict import CIMultiDict

from stremthru.error import StremThruError
from stremthru.version import VERSION

USER_AGENT = f"stremthru:sdk:py/{VERSION}"

Data = TypeVar("Data")


class ResponseMeta(TypedDict):
    headers: CIMultiDict[str]
    status_code: int


@dataclass
class Response(Generic[Data]):
    data: Data
    meta: ResponseMeta


class HealthData(TypedDict):
    status: Literal["ok"]


StremThruConfigAuthUserPass = dict[Literal["user", "pass"], str]
StremThruConfigAuthStoreToken = dict[Literal["store", "token"], str]
StremThruConfigAuth = Union[
    str, StremThruConfigAuthUserPass, StremThruConfigAuthStoreToken
]


class StremThru:
    base_url: str
    _headers: dict[str, str] = {}
    _timeout: aiohttp.ClientTimeout | None = None

    def __init__(
        self,
        base_url: str,
        auth: StremThruConfigAuth | None = None,
        user_agent: str | None = None,
        timeout: int | None = None,
        client_ip: str | None = None,
    ) -> None:
        self.base_url = base_url
        self._headers["User-Agent"] = (
            f"{USER_AGENT} {user_agent}" if user_agent else USER_AGENT
        )
        if auth:
            if isinstance(auth, dict) and "user" in auth:
                auth = cast(StremThruConfigAuthUserPass, auth)
                auth = f"{auth.get('user')}:{auth.get('pass')}"

            if isinstance(auth, str):
                if ":" in auth:
                    auth = base64.b64encode(auth.strip().encode()).decode()
                self._headers["X-StremThru-Authorization"] = f"Basic {auth}"
            elif "store" in auth:
                auth = cast(StremThruConfigAuthStoreToken, auth)
                self._headers["X-StremThru-Store-Name"] = auth.get("store", "")
                self._headers["X-StremThru-Store-Authorization"] = (
                    f"Bearer {auth.get('token')}"
                )
        if timeout:
            self._timeout = aiohttp.ClientTimeout(total=timeout)

        self.store = StremThruStore(self, client_ip)

    async def health(self) -> Response[HealthData]:
        return await self.request("/v0/health")

    async def request(
        self,
        endpoint: str,
        method: str = "GET",
        headers: Optional[dict[str, str]] = None,
        params: Optional[Mapping[str, Any]] = None,
        data: Optional[Mapping[str, Any] | aiohttp.FormData] = None,
        json: Optional[dict[str, Any]] = None,
    ) -> Response[Any]:
        url = f"{self.base_url}{endpoint}"
        headers = {
            "accept": "*/*",
            "accept-encoding": "gzip, deflate",
            **self._headers,
            **(headers or {}),
        }

        async with aiohttp.ClientSession() as client:
            async with client.request(
                method,
                url,
                headers=headers,
                params=params,
                data=data,
                json=json,
                timeout=self._timeout,
            ) as res:
                status_code = res.status

                if "application/json" in res.content_type:
                    res_data = await res.json()
                else:
                    res_data = await res.text()

                meta = ResponseMeta(
                    headers=res.headers.copy(),
                    status_code=status_code,
                )

                if not res.ok:
                    raise StremThruError(
                        body=res_data,
                        headers=meta["headers"],
                        status_code=meta["status_code"],
                    )

                return Response(
                    data=res_data.get("data", None)
                    if isinstance(res_data, dict)
                    else None,
                    meta=meta,
                )


StoreMagnetStatus = Literal[
    "cached",
    "downloaded",
    "downloading",
    "failed",
    "invalid",
    "processing",
    "queued",
    "unknown",
    "uploading",
]

StoreNewzStatus = Literal[
    "cached",
    "downloaded",
    "downloading",
    "failed",
    "invalid",
    "processing",
    "queued",
    "unknown",
]

StoreTorzStatus = Literal[
    "cached",
    "downloaded",
    "downloading",
    "failed",
    "invalid",
    "processing",
    "queued",
    "unknown",
    "uploading",
]


class AddMagnetDataFile(TypedDict):
    index: int
    link: str
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class AddMagnetData(TypedDict):
    added_at: str
    files: list[AddMagnetDataFile]
    hash: str
    id: str
    magnet: str
    name: str
    private: Optional[bool]
    status: StoreMagnetStatus


class CheckMagnetDataItemFile(TypedDict):
    index: int
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class CheckMagnetDataItem(TypedDict):
    files: list[CheckMagnetDataItemFile]
    hash: str
    magnet: str
    status: StoreMagnetStatus


class CheckMagnetData(TypedDict):
    items: list[CheckMagnetDataItem]


class GenerateLinkData(TypedDict):
    link: str


class GetMagnetDataFile(TypedDict):
    index: int
    link: str
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class GetMagnetData(TypedDict):
    added_at: str
    files: list[GetMagnetDataFile]
    hash: str
    id: str
    name: str
    private: Optional[bool]
    status: StoreMagnetStatus


StoreUserSubscriptionStatus = Literal["expired", "premium", "trial"]


class GetUserData(TypedDict):
    email: str
    id: str
    subscription_status: StoreUserSubscriptionStatus


class ListMagnetsDataItem(TypedDict):
    added_at: str
    hash: str
    id: str
    name: str
    private: Optional[bool]
    status: StoreMagnetStatus


class ListMagnetsData(TypedDict):
    items: list[ListMagnetsDataItem]
    total_items: int


class AddNewzData(TypedDict):
    hash: str
    id: str
    status: StoreNewzStatus


class CheckNewzDataItemFile(TypedDict):
    index: int
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class CheckNewzDataItem(TypedDict):
    files: list[CheckNewzDataItemFile]
    hash: str
    status: StoreNewzStatus


class CheckNewzData(TypedDict):
    items: list[CheckNewzDataItem]


class GetNewzDataFile(TypedDict):
    index: int
    link: str
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class GetNewzData(TypedDict):
    added_at: str
    files: list[GetNewzDataFile]
    hash: str
    id: str
    name: str
    size: int
    status: StoreNewzStatus


class ListNewzDataItem(TypedDict):
    added_at: str
    hash: str
    id: str
    name: str
    size: int
    status: StoreNewzStatus


class ListNewzData(TypedDict):
    items: list[ListNewzDataItem]
    total_items: int


class AddTorzDataFile(TypedDict):
    index: int
    link: str
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class AddTorzData(TypedDict):
    added_at: str
    files: list[AddTorzDataFile]
    hash: str
    id: str
    magnet: str
    name: str
    private: Optional[bool]
    size: int
    status: StoreTorzStatus


class CheckTorzDataItemFile(TypedDict):
    index: int
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class CheckTorzDataItem(TypedDict):
    files: list[CheckTorzDataItemFile]
    hash: str
    magnet: str
    status: StoreTorzStatus


class CheckTorzData(TypedDict):
    items: list[CheckTorzDataItem]


class GetTorzDataFile(TypedDict):
    index: int
    link: str
    name: str
    path: str
    size: int
    video_hash: Optional[str]


class GetTorzData(TypedDict):
    added_at: str
    files: list[GetTorzDataFile]
    hash: str
    id: str
    name: str
    private: Optional[bool]
    size: int
    status: StoreTorzStatus


class ListTorzDataItem(TypedDict):
    added_at: str
    hash: str
    id: str
    name: str
    private: Optional[bool]
    size: int
    status: StoreTorzStatus


class ListTorzData(TypedDict):
    items: list[ListTorzDataItem]
    total_items: int


class StremThruStoreNewz:
    def __init__(self, client: StremThru):
        self.client = client

    async def add(
        self,
        link: Optional[str] = None,
        file: Optional[io.BufferedReader] = None,
    ) -> Response[AddNewzData]:
        if file is not None:
            data = aiohttp.FormData()
            data.add_field("file", file)
            return await self.client.request(
                "/v0/store/newz",
                "POST",
                data=data,
            )

        return await self.client.request(
            "/v0/store/newz",
            "POST",
            json={"link": link},
        )

    async def check(self, hash: list[str]) -> Response[CheckNewzData]:
        return await self.client.request("/v0/store/newz/check", params={"hash": hash})

    async def generate_link(self, link: str) -> Response[GenerateLinkData]:
        return await self.client.request(
            "/v0/store/newz/link/generate",
            "POST",
            json={"link": link},
        )

    async def get(self, newz_id: str) -> Response[GetNewzData]:
        return await self.client.request(f"/v0/store/newz/{newz_id}")

    async def list(
        self, limit: int | None = None, offset: int | None = None
    ) -> Response[ListNewzData]:
        params = {}
        if limit:
            params["limit"] = limit
        if offset:
            params["offset"] = offset
        return await self.client.request("/v0/store/newz", params=params)

    async def remove(self, newz_id: str) -> Response[None]:
        return await self.client.request(f"/v0/store/newz/{newz_id}", "DELETE")


class StremThruStoreTorz:
    def __init__(self, client: StremThru):
        self.client = client

    async def add(
        self,
        link: Optional[str] = None,
        file: Optional[io.BufferedReader] = None,
    ) -> Response[AddTorzData]:
        if file is not None:
            data = aiohttp.FormData()
            data.add_field("file", file)
            return await self.client.request(
                "/v0/store/torz",
                "POST",
                data=data,
            )

        return await self.client.request(
            "/v0/store/torz",
            "POST",
            json={"link": link},
        )

    async def check(
        self, hash: list[str], sid: Optional[str] = None
    ) -> Response[CheckTorzData]:
        params: dict[str, Any] = {"hash": hash}
        if sid:
            params["sid"] = sid
        return await self.client.request("/v0/store/torz/check", params=params)

    async def generate_link(self, link: str) -> Response[GenerateLinkData]:
        return await self.client.request(
            "/v0/store/torz/link/generate",
            "POST",
            json={"link": link},
        )

    async def get(self, torz_id: str) -> Response[GetTorzData]:
        return await self.client.request(f"/v0/store/torz/{torz_id}")

    async def list(
        self, limit: int | None = None, offset: int | None = None
    ) -> Response[ListTorzData]:
        params = {}
        if limit:
            params["limit"] = limit
        if offset:
            params["offset"] = offset
        return await self.client.request("/v0/store/torz", params=params)

    async def remove(self, torz_id: str) -> Response[None]:
        return await self.client.request(f"/v0/store/torz/{torz_id}", "DELETE")


class StremThruStore:
    _client_ip: str | None = None

    def __init__(self, client: StremThru, client_ip: str | None = None):
        self.client = client
        if client_ip:
            self._client_ip = client_ip
        self.newz = StremThruStoreNewz(client)
        self.torz = StremThruStoreTorz(client)

    async def add_magnet(
        self,
        magnet: Optional[str] = None,
        torrent: Optional[io.BufferedReader | str] = None,
        client_ip: str | None = None,
    ) -> Response[AddMagnetData]:
        if not client_ip:
            client_ip = self._client_ip

        if type(magnet) is str:
            return await self.client.request(
                "/v0/store/magnets",
                "POST",
                json={"magnet": magnet},
                params={"client_ip": client_ip} if client_ip else None,
            )

        if type(torrent) is str:
            return await self.client.request(
                "/v0/store/magnets",
                "POST",
                json={"torrent": torrent},
                params={"client_ip": client_ip} if client_ip else None,
            )

        data = aiohttp.FormData()
        data.add_field("torrent", torrent)
        return await self.client.request(
            "/v0/store/magnets",
            "POST",
            data=data,
            params={"client_ip": client_ip} if client_ip else None,
        )

    async def check_magnet(
        self, magnet: list[str], sid: Optional[str] = None
    ) -> Response[CheckMagnetData]:
        params: dict[str, Any] = {"magnet": magnet}
        if sid:
            params["sid"] = sid
        return await self.client.request("/v0/store/magnets/check", params=params)

    async def generate_link(
        self, link: str, client_ip: str | None = None
    ) -> Response[GenerateLinkData]:
        if not client_ip:
            client_ip = self._client_ip

        return await self.client.request(
            "/v0/store/link/generate",
            "POST",
            json={"link": link},
            params={"client_ip": client_ip} if client_ip else None,
        )

    async def get_magnet(self, magnet_id: str) -> Response[GetMagnetData]:
        return await self.client.request(f"/v0/store/magnets/{magnet_id}")

    async def get_user(self) -> Response[GetUserData]:
        return await self.client.request("/v0/store/user")

    async def list_magnets(
        self, limit: int | None = None, offset: int | None = None
    ) -> Response[ListMagnetsData]:
        params = {}
        if limit:
            params["limit"] = limit
        if offset:
            params["offset"] = offset
        return await self.client.request("/v0/store/magnets", params=params)

    async def remove_magnet(self, magnet_id: str) -> Response[None]:
        return await self.client.request(f"/v0/store/magnets/{magnet_id}", "DELETE")
