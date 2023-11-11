import { ComponentFixture, TestBed } from '@angular/core/testing';

import { HighLowHistoryComponent } from './high-low-history.component';

describe('HighLowHistoryComponent', () => {
  let component: HighLowHistoryComponent;
  let fixture: ComponentFixture<HighLowHistoryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      declarations: [HighLowHistoryComponent]
    });
    fixture = TestBed.createComponent(HighLowHistoryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
