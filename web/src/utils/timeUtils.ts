export const formatNextRun = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = date.getTime() - now.getTime();
  const diffMins = Math.round(diffMs / 60000);

  if (diffMins < 60) {
    return `in ${diffMins} minute${diffMins !== 1 ? "s" : ""}`;
  } else if (diffMins < 1440) {
    const hours = Math.floor(diffMins / 60);
    return `in ${hours} hour${hours !== 1 ? "s" : ""}`;
  } else {
    const days = Math.floor(diffMins / 1440);
    return `in ${days} day${days !== 1 ? "s" : ""}`;
  }
};
