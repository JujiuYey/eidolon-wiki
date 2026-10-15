import { onMounted, onUnmounted, ref, type Ref } from "vue";

const MOBILE_BREAKPOINT = 768;

export function useIsMobile(): Ref<boolean> {
  const isMobile = ref(false);

  const update = () => {
    isMobile.value = window.innerWidth < MOBILE_BREAKPOINT;
  };

  onMounted(() => {
    const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
    mql.addEventListener("change", update);
    update();
    onUnmounted(() => mql.removeEventListener("change", update));
  });

  return isMobile;
}
