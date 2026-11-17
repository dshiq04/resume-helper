import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEventHandler,
  type ReactNode,
  type Ref,
} from "react";
import { AutoComplete, Input } from "antd";
import type { InputRef } from "antd";
import { api } from "../api/client";

interface CompanyAutoCompleteProps {
  value?: string;
  onChange?: (value: string) => void;
  onSelect?: (value: string) => void;
  onBlur?: () => void;
  onPressEnter?: () => void;
  onKeyDown?: KeyboardEventHandler<HTMLInputElement>;
  autoFocus?: boolean;
  size?: "small" | "middle" | "large";
  maxLength?: number;
  placeholder?: string;
  prefix?: ReactNode;
  disabled?: boolean;
  allowClear?: boolean;
  className?: string;
  inputRef?: Ref<InputRef>;
}

function highlightMatch(text: string, keyword: string): ReactNode {
  const needle = keyword.trim();
  if (!needle) return text;
  const index = text.toLowerCase().indexOf(needle.toLowerCase());
  if (index < 0) return text;
  return (
    <>
      {text.slice(0, index)}
      <span className="company-match">{text.slice(index, index + needle.length)}</span>
      {text.slice(index + needle.length)}
    </>
  );
}

export function CompanyAutoComplete({
  value = "",
  onChange,
  onSelect,
  onBlur,
  onPressEnter,
  onKeyDown,
  autoFocus,
  size,
  maxLength = 255,
  placeholder,
  prefix,
  disabled,
  allowClear,
  className,
  inputRef,
}: CompanyAutoCompleteProps) {
  const [companies, setCompanies] = useState<string[]>([]);
  const [open, setOpen] = useState(false);
  const requestRef = useRef(0);
  const timerRef = useRef<number | null>(null);
  const blurTimerRef = useRef<number | null>(null);

  const load = useCallback((search: string) => {
    const request = ++requestRef.current;
    void api
      .companySuggestions(search)
      .then((result) => {
        if (request !== requestRef.current) return;
        setCompanies(result.items ?? []);
      })
      .catch(() => {
        if (request !== requestRef.current) return;
        setCompanies([]);
      });
  }, []);

  const scheduleLoad = useCallback(
    (search: string) => {
      if (timerRef.current) window.clearTimeout(timerRef.current);
      timerRef.current = window.setTimeout(() => load(search), 180);
    },
    [load],
  );

  useEffect(
    () => () => {
      if (timerRef.current) window.clearTimeout(timerRef.current);
      if (blurTimerRef.current) window.clearTimeout(blurTimerRef.current);
    },
    [],
  );

  const options = useMemo(
    () =>
      companies.map((company) => ({
        value: company,
        label: highlightMatch(company, value),
      })),
    [companies, value],
  );

  return (
    <AutoComplete
      className={className}
      value={value}
      options={options}
      open={open && options.length > 0}
      popupMatchSelectWidth={300}
      filterOption={false}
      onFocus={() => {
        setOpen(true);
        load(value);
      }}
      onDropdownVisibleChange={(visible) => {
        setOpen(visible);
        if (visible) load(value);
      }}
      onSearch={(search) => {
        scheduleLoad(search);
      }}
      onChange={(next) => {
        onChange?.(next);
        if (next.trim() === "") load("");
      }}
      onSelect={(next) => {
        onChange?.(next);
        onSelect?.(next);
      }}
    >
      <Input
        ref={inputRef}
        autoFocus={autoFocus}
        size={size}
        maxLength={maxLength}
        disabled={disabled}
        allowClear={allowClear}
        placeholder={placeholder}
        prefix={prefix}
        onFocus={() => {
          setOpen(true);
          load(value);
        }}
        onBlur={() => {
          if (blurTimerRef.current) window.clearTimeout(blurTimerRef.current);
          blurTimerRef.current = window.setTimeout(() => {
            setOpen(false);
            onBlur?.();
          }, 120);
        }}
        onPressEnter={() => onPressEnter?.()}
        onKeyDown={onKeyDown}
      />
    </AutoComplete>
  );
}
