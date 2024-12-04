import base64
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
                self._headers["Proxy-Authorization"] = f"Basic {auth}"
            elif "store" in auth:
                auth = cast(StremThruConfigAuthStoreToken, auth)
                self._headers["X-StremThru-Store-Name"] = auth.get("store", "")
                self._headers["X-StremThru-Store-Authorization"] = (
                    f"Bearer {auth.get('token')}"
                )
        if timeout:
            self._timeout = aiohttp.ClientTimeout(total=timeout)

        self.store = StremThruStore(self)

    async def health(self) -> Response[HealthData]:
        return await self.request("/v0/health")

    async def request(
        self,
        endpoint: str,
        method: str = "GET",
        headers: Optional[dict[str, str]] = None,
        params: Optional[Mapping[str, Any]] = None,
        data: Optional[Mapping[str, Any]] = None,
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


class AddMagnetDataFile(TypedDict):
    index: int
    link: str
    name: str
    path: str
    size: int


class AddMagnetData(TypedDict):
    files: list[AddMagnetDataFile]
    hash: str
    id: str
    magnet: str
    name: str
    status: StoreMagnetStatus


class CheckMagnetDataItemFile(TypedDict):
    index: int
    name: str
    size: int


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


class GetMagnetData(TypedDict):
    files: list[GetMagnetDataFile]
    hash: str
    id: str
    name: str
    status: StoreMagnetStatus


StoreUserSubscriptionStatus = Literal["expired", "premium", "trial"]


class GetUserData(TypedDict):
    email: str
    id: str
    subscription_status: StoreUserSubscriptionStatus


class ListMagnetsDataItem(TypedDict):
    hash: str
    id: str
    name: str
    status: StoreMagnetStatus


class ListMagnetsData(TypedDict):
    items: list[ListMagnetsDataItem]
    total_items: int


class StremThruStore:
    def __init__(self, client: StremThru):
        self.client = client

    async def add_magnet(self, magnet: str) -> Response[AddMagnetData]:
        return await self.client.request(
            "/v0/store/magnets", "POST", json={"magnet": magnet}
        )

    async def check_magnet(self, magnet: list[str]) -> Response[CheckMagnetData]:
        return await self.client.request("/v0/store/magnets", params={"magnet": magnet})

    async def generate_link(self, link: str) -> Response[GenerateLinkData]:
        return await self.client.request(
            "/v0/store/link/generate", "POST", json={"link": link}
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
