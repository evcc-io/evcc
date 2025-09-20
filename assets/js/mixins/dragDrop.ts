/**
 * Drag and Drop Mixin for Vue components
 *
 * Provides functionality for reordering items via mouse drag or touch gestures.
 *
 * Usage:
 * 1. Import and add to component mixins
 * 2. Set data properties: draggedIndex, touchStartY, touchStartTime, isDragging
 * 3. Implement handleReorder(fromIndex: number, toIndex: number) method
 * 4. Use provided event handlers in template
 * 5. Use provided CSS classes for styling
 */

export interface DragDropMixin {
  // Data properties that must be defined in the component
  draggedIndex: number;
  touchStartY: number;
  touchStartTime: number;
  isDragging: boolean;

  // Method that must be implemented in the component
  handleReorder(fromIndex: number, toIndex: number): void;

  // Provided methods
  onDragStart(index: number, event: DragEvent): void;
  onDrop(targetIndex: number, event: DragEvent): void;
  onTouchStart(index: number, event: TouchEvent): void;
  onTouchMove(event: TouchEvent, itemSelector?: string): void;
  onTouchEnd(event: TouchEvent): void;
  isItemBeingDragged(index: number): boolean;
  resetDragState(): void;
}

export default {
  methods: {
    /**
     * Handle desktop drag start
     */
    onDragStart(index: number, event: DragEvent) {
      (this as any).draggedIndex = index;
      if (event.dataTransfer) {
        event.dataTransfer.effectAllowed = "move";
        event.dataTransfer.setData("text/plain", index.toString());
      }
    },

    /**
     * Handle desktop drop
     */
    onDrop(targetIndex: number, event: DragEvent) {
      event.preventDefault();
      if ((this as any).draggedIndex === -1 || (this as any).draggedIndex === targetIndex) {
        return;
      }

      (this as any).handleReorder((this as any).draggedIndex, targetIndex);
      (this as any).draggedIndex = -1;
    },

    /**
     * Handle touch start for mobile devices
     */
    onTouchStart(index: number, event: TouchEvent) {
      (this as any).draggedIndex = index;
      (this as any).touchStartY = event.touches[0]?.clientY || 0;
      (this as any).touchStartTime = Date.now();
      (this as any).isDragging = false;
    },

    /**
     * Handle touch move for mobile devices
     * @param event - Touch event
     * @param itemSelector - CSS selector for draggable items (defaults to current implementation)
     * @param touchThreshold - Minimum pixels to start drag (default: 10)
     * @param timeThreshold - Minimum time to start drag (default: 200ms)
     */
    onTouchMove(
      event: TouchEvent,
      itemSelector = ".drag-drop-item",
      touchThreshold = 10,
      timeThreshold = 200
    ) {
      if ((this as any).draggedIndex === -1) return;

      event.preventDefault();
      const touch = event.touches[0];
      if (!touch) return;

      const moveDistance = Math.abs(touch.clientY - (this as any).touchStartY);
      const moveTime = Date.now() - (this as any).touchStartTime;

      // Start dragging if moved more than threshold or after time threshold
      if (
        !(this as any).isDragging &&
        (moveDistance > touchThreshold || moveTime > timeThreshold)
      ) {
        (this as any).isDragging = true;
      }

      if ((this as any).isDragging) {
        // Find the element under the touch point
        const elementBelow = document.elementFromPoint(touch.clientX, touch.clientY);
        const targetItem = elementBelow?.closest(itemSelector);

        if (targetItem && (this as any).$el) {
          const items = Array.from((this as any).$el.querySelectorAll(itemSelector));
          const targetIndex = items.indexOf(targetItem);

          if (targetIndex !== -1 && targetIndex !== (this as any).draggedIndex) {
            (this as any).handleReorder((this as any).draggedIndex, targetIndex);
            (this as any).draggedIndex = targetIndex;
          }
        }
      }
    },

    /**
     * Handle touch end for mobile devices
     */
    onTouchEnd(event: TouchEvent) {
      if ((this as any).draggedIndex === -1) return;

      // If we weren't dragging, treat it as a tap (don't prevent default)
      if (!(this as any).isDragging) {
        (this as any).draggedIndex = -1;
        return;
      }

      event.preventDefault();
      (this as any).draggedIndex = -1;
      (this as any).isDragging = false;
    },

    /**
     * Check if an item is currently being dragged
     */
    isItemBeingDragged(index: number): boolean {
      return (this as any).draggedIndex === index;
    },

    /**
     * Reset drag state (useful for cleanup)
     */
    resetDragState() {
      (this as any).draggedIndex = -1;
      (this as any).isDragging = false;
    },
  },
};
