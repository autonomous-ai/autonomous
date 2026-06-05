from typing import TYPE_CHECKING, Callable

if TYPE_CHECKING:
    import cv2


type SendEventCallable = Callable[
    [str, str, str, list[cv2.typing.MatLike] | None, float | None], None
]

type OnMotionCallable = Callable[[], None]
