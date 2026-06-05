const ONBOARDING_COMPLETE_KEY = "onboarding_complete";

export function useOnboardingComplete(): boolean {
  return typeof window !== "undefined" && localStorage.getItem(ONBOARDING_COMPLETE_KEY) === "true";
}

export function setOnboardingComplete(): void {
  if (typeof window !== "undefined") {
    localStorage.setItem(ONBOARDING_COMPLETE_KEY, "true");
  }
}

export function clearOnboardingComplete(): void {
  if (typeof window !== "undefined") {
    localStorage.removeItem(ONBOARDING_COMPLETE_KEY);
  }
}
