// https://github.com/elastic/eui/issues/5463
import { appendIconComponentCache } from '@elastic/eui/es/components/icon/icon';
import { icon as arrowDown } from '@elastic/eui/es/components/icon/assets/arrow_down';
import { icon as arrowUp } from '@elastic/eui/es/components/icon/assets/arrow_up';
import { icon as arrowLeft } from '@elastic/eui/es/components/icon/assets/arrow_left';
import { icon as arrowRight } from '@elastic/eui/es/components/icon/assets/arrow_right';
import { icon as check } from '@elastic/eui/es/components/icon/assets/check';
import { icon as cross } from '@elastic/eui/es/components/icon/assets/cross';
import { icon as empty } from '@elastic/eui/es/components/icon/assets/empty';
import { icon as help } from '@elastic/eui/es/components/icon/assets/help';
import { icon as logoElastic } from '@elastic/eui/es/components/icon/assets/logo_elastic';
import { icon as search } from '@elastic/eui/es/components/icon/assets/search';
import { icon as sortable } from '@elastic/eui/es/components/icon/assets/sortable';
import { icon as sortDown } from '@elastic/eui/es/components/icon/assets/sort_down';
import { icon as sortUp } from '@elastic/eui/es/components/icon/assets/sort_up';
import { icon as user } from '@elastic/eui/es/components/icon/assets/user';
import { icon as warning } from '@elastic/eui/es/components/icon/assets/warning';
import { icon as alert } from '@elastic/eui/es/components/icon/assets/alert';
import { icon as refresh } from '@elastic/eui/es/components/icon/assets/refresh';

// Register all icons in the component cache for static usage
appendIconComponentCache({
  arrowDown,
  arrowUp,
  arrowLeft,
  arrowRight,
  check,
  cross,
  empty,
  help,
  logoElastic,
  refresh,
  search,
  sortable,
  sortDown,
  sortUp,
  user,
  alert,
  warning,
});
