import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { CustomerBulkDeleteButton } from '@/pages/commercial/CommercialPage';

describe('CustomerBulkDeleteButton', () => {
  it('stays visible and explains that a customer must be selected', () => {
    const onDelete = vi.fn();
    const onEmpty = vi.fn();

    render(<CustomerBulkDeleteButton selectedCount={0} onDelete={onDelete} onEmpty={onEmpty} />);

    fireEvent.click(screen.getByRole('button', { name: '批量删除客户' }));

    expect(onEmpty).toHaveBeenCalledOnce();
    expect(onDelete).not.toHaveBeenCalled();
  });

  it('shows the selected count and starts deletion', () => {
    const onDelete = vi.fn();
    const onEmpty = vi.fn();

    render(<CustomerBulkDeleteButton selectedCount={3} onDelete={onDelete} onEmpty={onEmpty} />);

    fireEvent.click(screen.getByRole('button', { name: '批量删除已选择的 3 个客户' }));

    expect(screen.getByText('批量删除（3）')).toBeTruthy();
    expect(onDelete).toHaveBeenCalledOnce();
    expect(onEmpty).not.toHaveBeenCalled();
  });
});
